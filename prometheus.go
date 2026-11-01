package main

import (
	"monitordatamanager/tlsutil"

	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"bufio"
	"strconv"
	"strings"
	"unicode"

	"github.com/gocql/gocql"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	pcm "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

// Target represents the configuration of the target for the survey
type Target struct {
	URL      string            `yaml:"url"`
	Name     string            `yaml:"name"`
	Timeout  time.Duration     `yaml:"timeout"`
	Interval time.Duration     `yaml:"interval"`
	Labels   map[string]string `yaml:"labels"`
}

// MetricData represents a porrosa metric
type MetricData struct {
	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels"`
	Value     float64           `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
	Type      string            `json:"type"`
	Help      string            `json:"help,omitempty"`
	Target    string            `json:"target,omitempty"`
}

// Metricparser represents parser metric Prometheus
type MetricParser struct {
	includeHelp       bool
	includeTimestamps bool
}

// Global metric for monitoring scrapers
var (
	scrapeRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scraper_requests_total",
			Help: "Total number of scrape requests",
		},
		[]string{"target", "status"},
	)

	scrapeDuration = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "scraper_request_duration_seconds",
			Help: "Duration of scrape HTTP requests in seconds",
		},
		[]string{"target"},
	)

	scrapeDurationFull = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "scraper_request_duration_full_seconds",
			Help: "Duration of scrape HTTP requests + INSERT DB in seconds",
		},
		[]string{"target"},
	)

	scrapeErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scraper_errors_total",
			Help: "Total number of scrape errors",
		},
		[]string{"target", "error_type"},
	)

	lastSuccessfullScrape = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "scraper_last_success_timestamp",
			Help: "Unix timestamp of last successful scrape",
		},
		[]string{"target"},
	)

	activePrometheusWorkers = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "scraper_active_workers",
			Help: "Number of active workers",
		},
		[]string{},
	)
)

type Scraper struct {
	client        *http.Client
	parsedMetrics chan map[string][]MetricData
	wg            sync.WaitGroup
}

func NewScraper() *Scraper {
	var transport *http.Transport
	if AppConfig.PRMTSCRAPER.TLS {
		// crt, err := tls.LoadX509KeyPair(AppConfig.TLS.CERTPATH, AppConfig.TLS.KEYPATH)
		crt, err := tlsutil.LoadX509KeyPairWithPassword(AppConfig.TLS.CERTPATH, AppConfig.TLS.KEYPATH, []byte(AppConfig.TLS.KEYPASSWORD))
		if err != nil {
			log.Fatal("Failed to download the certificate (", err, ")!")
			transport = &http.Transport{
				MaxIdleConns:        AppConfig.PRMTSCRAPER.MAXCONNECTIONS,
				MaxIdleConnsPerHost: AppConfig.PRMTSCRAPER.MAXCONNECTIONS,
				IdleConnTimeout:     90 * time.Second,
			}
		} else {
			var crtPool *x509.CertPool
			if AppConfig.TLS.CAPATH == "" {
				crtPool = nil
			} else {
				crtPool = x509.NewCertPool()
				if crtCA, err := os.ReadFile(AppConfig.TLS.CAPATH); err != nil {
					log.Fatal("Failed to download CA certificate (", err, ")!")
				} else if ok := crtPool.AppendCertsFromPEM(crtCA); !ok {
					log.Fatal("It was not possible to apply CA certificate!")
				}
			}

			transport = &http.Transport{
				TLSClientConfig: &tls.Config{
					Certificates:       []tls.Certificate{crt},
					RootCAs:            crtPool,
					InsecureSkipVerify: AppConfig.PRMTSCRAPER.SKIPVERIFY,
					MinVersion:         tls.VersionTLS12,
				},
				MaxIdleConns:        AppConfig.PRMTSCRAPER.MAXCONNECTIONS,
				MaxIdleConnsPerHost: AppConfig.PRMTSCRAPER.MAXCONNECTIONS,
				IdleConnTimeout:     90 * time.Second,
			}
		}
	} else {
		transport = &http.Transport{
			MaxIdleConns:        AppConfig.PRMTSCRAPER.MAXCONNECTIONS,
			MaxIdleConnsPerHost: AppConfig.PRMTSCRAPER.MAXCONNECTIONS,
			IdleConnTimeout:     90 * time.Second,
		}
	}

	return &Scraper{
		client: &http.Client{
			Transport: transport,
			Timeout:   AppConfig.PRMTSCRAPER.TIMEOUT,
		},
		parsedMetrics: make(chan map[string][]MetricData, 100),
	}
}

func (s *Scraper) workersStart() {
	log.Printf("PROMETHEUS WORKERS START %d... %s", AppConfig.PRMTSCRAPER.WORKERSPOOL, "OK")

	for worker := 1; worker <= AppConfig.PRMTSCRAPER.WORKERSPOOL; worker++ {
		s.wg.Add(1)
		go s.scrapeTarget(worker)
	}
}

func (s *Scraper) Stop() {
	log.Println("Stopping Scraper...")
	s.wg.Wait()
	close(s.parsedMetrics)
	log.Println("Scraper stopped")
}

// scrapeTarget Relays a specific goal
func (s *Scraper) scrapeTarget(worker int) {

	var scraperUUID gocql.UUID
	var err error
	if AppConfig.DM.ID == "" {
		scraperUUID, err = gocql.RandomUUID()
		if err != nil {
			log.Printf("Failed to generate UUID for Scraper (%s)", err.Error())
			return
		}
	} else {
		scraperUUID, err = gocql.ParseUUID(AppConfig.DM.ID)
		if err != nil {
			log.Printf("Failed to parse UUID for Scraper (%s)", err.Error())
			return
		}
	}

	log.Printf("START PROMETHEUS WORKER %s... OK", scraperUUID)

	activePrometheusWorkers.WithLabelValues().Inc()
	defer activePrometheusWorkers.WithLabelValues().Dec()
	defer s.wg.Done()

	ticker := time.NewTicker(AppConfig.PRMTSCRAPER.INTERVAL)
	defer ticker.Stop()
	for {
		select {
		case <-SignalChan:
			log.Printf("PROMETHEUS WORKER %s STOP... OK", scraperUUID)
			return
		case <-ticker.C:
			ctx := context.Background()
			scanner := Session.Query(`SELECT module_id, space_id, status, metric, object, data_type, ol, ol_time, dm_id FROM task`).WithContext(ctx).Consistency(ConsistencyRead).Iter().Scanner()
			for scanner.Next() {
				var moduleID, spaceID, ol gocql.UUID
				var dmID *gocql.UUID
				var status int
				var metric, object, dataType string
				var olTime int64
				err := scanner.Scan(&moduleID, &spaceID, &status, &metric, &object, &dataType, &ol, &olTime, &dmID)
				if err != nil {
					log.Print(err)
				} else {
					if status == 1 && strings.ToUpper(object) == "PROMETHEUS" && strings.ToUpper(metric) == "ALL" && dmID == nil {
						log.Print("OPTIMISTIC BLOCK TASK...")
						spaceStatus := 0
						spaceDescription := ""
						ctx := context.Background()
						if err := Session.Query(`SELECT status, description FROM space WHERE id = ? LIMIT 1`, spaceID).WithContext(ctx).Consistency(ConsistencyRead).Scan(&spaceStatus, &spaceDescription); err != nil {
							log.Print(err)
						}
						if spaceStatus == 1 {
							var applied bool
							var err error

							ctx := context.Background()
							applied, err = Session.Query(`UPDATE task SET ol = ?, ol_time = toTimestamp(now()) WHERE module_id = ? AND space_id = ? AND object = ? AND metric = ? IF ol_time < (toTimestamp(now()) - (bigint) ?)`,
								scraperUUID, moduleID, spaceID, object, metric, AppConfig.PRMTSCRAPER.INTERVAL.Milliseconds()).WithContext(ctx).ScanCAS(nil)
							if err != nil {
								log.Printf("Failed to update task (%s)", err.Error())
							} else {
								if applied {
									log.Printf("Optimistic block %d (%s)... OK", makeTimestamp(), moduleID)

									moduleAddress := ""
									moduleStatus := 0
									ctx := context.Background()
									if err := Session.Query(`SELECT status, address FROM registration WHERE id = ? LIMIT 1`, moduleID).WithContext(ctx).Consistency(ConsistencyRead).Scan(&moduleStatus, &moduleAddress); err != nil {
										log.Print(err)
									}
									if moduleStatus == 1 {
										start := time.Now()

										for attempt := 1; attempt <= AppConfig.PRMTSCRAPER.RETRYATTEMPTS; attempt++ {
											metrics, err := s.scrapeWithRetry(moduleAddress, moduleID.String())
											if err == nil {
												// Successful Scrape
												scrapeRequestsTotal.WithLabelValues(moduleID.String(), "success").Inc()
												lastSuccessfullScrape.WithLabelValues(moduleID.String()).Set(float64(time.Now().Unix()))
												duration := time.Since(start).Seconds()
												scrapeDuration.WithLabelValues(moduleID.String()).Set(duration)

												log.Print("Scraper получил ", len(metrics), " метрик от ", moduleID, ", DATA SAVING...")
												// ctx := context.Background()

												for i := 0; i < len(metrics); i += AppConfig.DB.BATCHSIZE {
													end := min(i+AppConfig.DB.BATCHSIZE, len(metrics))

													batch := Session.NewBatch(gocql.LoggedBatch)
													for j := i; j < end; j++ {
														batch.Query(`INSERT INTO raw_data (metric, value, create_time, event_time, space_id, object, space_description, labels) VALUES (?, ?, toTimestamp(now()), toTimestamp(now()), ?, 'prometheus', ?, ?)`,
															metrics[j].Name, float32(metrics[j].Value), spaceID, spaceDescription, metrics[j].Labels)
														batch.Query(`INSERT INTO space_metric_data (space_id, metric) VALUES (?, ?)`,
															spaceID, metrics[j].Name)
													}
													if err := Session.ExecuteBatch(batch); err != nil {
														log.Printf("Scrapper ERROR EXECUTE BATCH %v", err)
													}
												}

												err := rawCount(spaceID, "data", len(metrics))
												if err != nil {
													log.Print(err)
												}

												durationFull := time.Since(start).Seconds()
												scrapeDurationFull.WithLabelValues(moduleID.String()).Set(durationFull)
												log.Print("Scraper ", moduleID, " DATA SAVED... OK")
												break
											} else {
												if attempt < AppConfig.PRMTSCRAPER.RETRYATTEMPTS {
													log.Printf("%d Attempt %d failed for %s: %v, retrying in %v",
														worker, attempt, moduleID, err, AppConfig.PRMTSCRAPER.RETRYDELAY)
													time.Sleep(AppConfig.PRMTSCRAPER.RETRYDELAY)
												} else {
													// All attempts by Failed
													scrapeRequestsTotal.WithLabelValues(moduleID.String(), "error").Inc()
													scrapeErrors.WithLabelValues(moduleID.String(), "scrape_error").Inc()
													log.Printf("%d All attempts failed for %s: %v", worker, moduleID, err)
													break
												}
											}
										}
									}
								} else {
									log.Printf("Optimistic block %d (%s)... FAILED", makeTimestamp(), moduleID)
								}
							}
						}
					} else if status == 1 && strings.ToUpper(object) == "PROMETHEUS" && strings.ToUpper(metric) == "ALL" && dmID.String() == AppConfig.DM.ID {
						log.Print("MY TASK...")
						spaceStatus := 0
						spaceDescription := ""
						ctx := context.Background()
						if err := Session.Query(`SELECT status, description FROM space WHERE id = ? LIMIT 1`, spaceID).WithContext(ctx).Consistency(ConsistencyRead).Scan(&spaceStatus, &spaceDescription); err != nil {
							log.Print(err)
						}
						if spaceStatus == 1 {

							moduleAddress := ""
							moduleStatus := 0
							ctx := context.Background()
							if err := Session.Query(`SELECT status, address FROM registration WHERE id = ? LIMIT 1`, moduleID).WithContext(ctx).Consistency(ConsistencyRead).Scan(&moduleStatus, &moduleAddress); err != nil {
								log.Print(err)
							}
							if moduleStatus == 1 {
								start := time.Now()

								for attempt := 1; attempt <= AppConfig.PRMTSCRAPER.RETRYATTEMPTS; attempt++ {
									metrics, err := s.scrapeWithRetry(moduleAddress, moduleID.String())
									if err == nil {
										// Successful Scrape
										scrapeRequestsTotal.WithLabelValues(moduleID.String(), "success").Inc()
										lastSuccessfullScrape.WithLabelValues(moduleID.String()).Set(float64(time.Now().Unix()))
										duration := time.Since(start).Seconds()
										scrapeDuration.WithLabelValues(moduleID.String()).Set(duration)

										log.Print("Scraper получил ", len(metrics), " метрик от ", moduleID, ", DATA SAVING...")
										for i := 0; i < len(metrics); i += AppConfig.DB.BATCHSIZE {
											end := min(i+AppConfig.DB.BATCHSIZE, len(metrics))

											batch := Session.NewBatch(gocql.LoggedBatch)
											for j := i; j < end; j++ {
												batch.Query(`INSERT INTO raw_data (metric, value, create_time, event_time, space_id, object, space_description, labels) VALUES (?, ?, toTimestamp(now()), toTimestamp(now()), ?, 'prometheus', ?, ?)`,
													metrics[j].Name, float32(metrics[j].Value), spaceID, spaceDescription, metrics[j].Labels)
												batch.Query(`INSERT INTO space_metric_data (space_id, metric) VALUES (?, ?)`,
													spaceID, metrics[j].Name)
											}
											if err := Session.ExecuteBatch(batch); err != nil {
												log.Printf("Scrapper ERROR EXECUTE BATCH %v", err)
											}
										}

										err := rawCount(spaceID, "data", len(metrics))
										if err != nil {
											log.Print(err)
										}

										durationFull := time.Since(start).Seconds()
										scrapeDurationFull.WithLabelValues(moduleID.String()).Set(durationFull)
										log.Print("Scraper ", moduleID, " DATA SAVED... OK")
										break
									} else {
										if attempt < AppConfig.PRMTSCRAPER.RETRYATTEMPTS {
											log.Printf("%d Attempt %d failed for %s: %v, retrying in %v",
												worker, attempt, moduleID, err, AppConfig.PRMTSCRAPER.RETRYDELAY)
											time.Sleep(AppConfig.PRMTSCRAPER.RETRYDELAY)
										} else {
											// All attempts by Failed
											scrapeRequestsTotal.WithLabelValues(moduleID.String(), "error").Inc()
											scrapeErrors.WithLabelValues(moduleID.String(), "scrape_error").Inc()
											log.Printf("%d All attempts failed for %s: %v", worker, moduleID, err)
											break
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// scrapeWithRetry performs a survey with error processing
func (s *Scraper) scrapeWithRetry(address string, moduleID string) ([]MetricData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(30)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", address, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "MONITORDataManager")
	req.Header.Set("Accept", "text/plain")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	// Parsing metrics
	metrics, err := s.parseMetricsFromResponse(resp, moduleID)
	if err != nil {
		return nil, fmt.Errorf("parse metrics: %w", err)
	}

	return metrics, nil
}

// parseMetricsFromResponse Pars the metric from http response
func (s *Scraper) parseMetricsFromResponse(resp *http.Response, targetName string) ([]MetricData, error) {
	parser := NewMetricParser(true, true)

	// We use the official Prometheus parser
	metrics, err := parser.ParseMetricsWithOfficialParser(resp.Body)
	if err != nil {
		log.Printf("Prometheus parser failed, falling back to custom parser: %v", err)

		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("read body: %w", readErr)
		}

		metrics, err = parser.ParseMetrics(string(body))
		if err != nil {
			return nil, fmt.Errorf("both parsers failed: %w", err)
		}
	}

	for i := range metrics {
		metrics[i].Target = targetName
	}

	return metrics, nil
}

// GetResults Returns a channel with a survey results
func (s *Scraper) GetResults() <-chan map[string][]MetricData {
	return s.parsedMetrics
}

// NewMetricParser Creates a new parser
func NewMetricParser(includeHelp, includeTimestamps bool) *MetricParser {
	return &MetricParser{
		includeHelp:       includeHelp,
		includeTimestamps: includeTimestamps,
	}
}

// ParseMetrics Parses metric from the text using the official parser
func (p *MetricParser) ParseMetricsWithOfficialParser(reader io.Reader) ([]MetricData, error) {
	parser := expfmt.NewTextParser(model.UTF8Validation)
	metricFamilies, err := parser.TextToMetricFamilies(reader)
	if err != nil {
		return nil, fmt.Errorf("official parser failed: %w", err)
	}

	var metrics []MetricData
	now := time.Now()

	for familyName, family := range metricFamilies {
		for _, metric := range family.Metric {
			metricData := p.convertToMetricData(family, metric, familyName, now)
			metrics = append(metrics, metricData)
		}
	}

	return metrics, nil
}

// convertToMetricData converts the metric to our format
func (p *MetricParser) convertToMetricData(family *pcm.MetricFamily, metric *pcm.Metric, familyName string, timestamp time.Time) MetricData {
	labels := make(map[string]string)

	// Extracting labels
	for _, labelPair := range metric.Label {
		if labelPair.Name != nil && labelPair.Value != nil {
			labels[*labelPair.Name] = *labelPair.Value
		}
	}

	// Extracting the value
	var value float64
	if metric.Counter != nil && metric.Counter.Value != nil {
		value = *metric.Counter.Value
	} else if metric.Gauge != nil && metric.Gauge.Value != nil {
		value = *metric.Gauge.Value
	} else if metric.Histogram != nil && metric.Histogram.SampleCount != nil {
		value = float64(*metric.Histogram.SampleCount)
	} else if metric.Summary != nil && metric.Summary.SampleCount != nil {
		value = float64(*metric.Summary.SampleCount)
	} else if metric.Untyped != nil && metric.Untyped.Value != nil {
		value = *metric.Untyped.Value
	}

	// Determining the type of metric
	metricType := "untyped"
	if family.Type != nil {
		switch *family.Type {
		case pcm.MetricType_COUNTER:
			metricType = "counter"
		case pcm.MetricType_GAUGE:
			metricType = "gauge"
		case pcm.MetricType_HISTOGRAM:
			metricType = "histogram"
		case pcm.MetricType_SUMMARY:
			metricType = "summary"
		}
	}

	// Setting the timestamp
	var metricTimestamp time.Time
	if p.includeTimestamps && metric.TimestampMs != nil {
		metricTimestamp = time.Unix(0, *metric.TimestampMs*int64(time.Millisecond))
	} else {
		metricTimestamp = timestamp
	}

	return MetricData{
		Name:      familyName,
		Labels:    labels,
		Value:     value,
		Timestamp: metricTimestamp,
		Type:      metricType,
		Help:      p.getHelp(family),
	}
}

// getHelp returns help text if enabled
func (p *MetricParser) getHelp(family *pcm.MetricFamily) string {
	if p.includeHelp && family.Help != nil {
		return *family.Help
	}
	return ""
}

// ParseMetrics parses metrics using a custom parser
func (p *MetricParser) ParseMetrics(metricsText string) ([]MetricData, error) {
	var metrics []MetricData
	scanner := bufio.NewScanner(strings.NewReader(metricsText))

	var currentHelp, currentType string
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}

		// Parsim HELP comments
		if strings.HasPrefix(line, "# HELP ") {
			if p.includeHelp {
				currentHelp = p.parseHelpLine(line)
			}
			continue
		}

		// Parsim TYPE comments
		if strings.HasPrefix(line, "# TYPE ") {
			currentType = p.parseTypeLine(line)
			continue
		}

		// Parsing metrics
		if !strings.HasPrefix(line, "#") {
			metric, err := p.parseMetricLine(line, currentType, currentHelp)
			if err != nil {
				log.Printf("Error parsing line %d: %v - %s", lineNumber, err, line)
				continue
			}
			metrics = append(metrics, metric)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return metrics, nil
}

// parseHelpLine parses the HELP string
func (p *MetricParser) parseHelpLine(line string) string {
	parts := strings.Fields(line)
	if len(parts) >= 4 {
		return strings.Join(parts[3:], " ")
	}
	return ""
}

// parseTypeLine парсит TYPE строку
func (p *MetricParser) parseTypeLine(line string) string {
	parts := strings.Fields(line)
	if len(parts) >= 4 {
		return parts[3]
	}
	return "untyped"
}

// parseMetricLine парсит строку с метрикой
func (p *MetricParser) parseMetricLine(line, metricType, help string) (MetricData, error) {
	// Разделяем на имя метрики + labels и значение
	parts := strings.Split(line, " ")
	if len(parts) < 2 {
		return MetricData{}, fmt.Errorf("invalid metric line format")
	}

	metricPart := strings.TrimSpace(parts[0])
	valuePart := strings.TrimSpace(parts[1])

	// Парсим значение
	value, err := strconv.ParseFloat(valuePart, 64)
	if err != nil {
		return MetricData{}, fmt.Errorf("invalid value '%s': %w", valuePart, err)
	}

	// Парсим timestamp если есть
	var timestamp time.Time
	if len(parts) >= 3 {
		if ts, err := strconv.ParseInt(parts[2], 10, 64); err == nil {
			timestamp = time.Unix(ts/1000, (ts%1000)*int64(time.Millisecond))
		}
	}

	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	// Парсим имя и labels
	name, labels, err := p.parseMetricNameAndLabels(metricPart)
	if err != nil {
		return MetricData{}, err
	}

	return MetricData{
		Name:      name,
		Labels:    labels,
		Value:     value,
		Timestamp: timestamp,
		Type:      metricType,
		Help:      help,
	}, nil
}

// parseMetricNameAndLabels парсит имя метрики и labels
func (p *MetricParser) parseMetricNameAndLabels(metricPart string) (string, map[string]string, error) {
	labels := make(map[string]string)

	if !strings.Contains(metricPart, "{") {
		// Метрика без labels
		return metricPart, labels, nil
	}

	// Извлекаем имя метрики
	braceIndex := strings.Index(metricPart, "{")
	name := metricPart[:braceIndex]

	// Извлекаем часть с labels
	labelsPart := metricPart[braceIndex+1 : len(metricPart)-1]
	if labelsPart == "" {
		return name, labels, nil
	}

	// Парсим labels
	currentLabel := ""
	currentValue := ""
	inQuotes := false
	escapeNext := false

	for _, char := range labelsPart {
		if escapeNext {
			currentValue += string(char)
			escapeNext = false
			continue
		}

		switch {
		case char == '\\':
			escapeNext = true
		case char == '"':
			inQuotes = !inQuotes
		case char == '=' && !inQuotes:
			if currentLabel != "" {
				return "", nil, fmt.Errorf("invalid labels format")
			}
			currentLabel = currentValue
			currentValue = ""
		case char == ',' && !inQuotes:
			if currentLabel != "" && currentValue != "" {
				labels[currentLabel] = currentValue
				currentLabel = ""
				currentValue = ""
			}
		case unicode.IsSpace(char) && !inQuotes:
			// Пропускаем пробелы вне кавычек
			continue
		default:
			currentValue += string(char)
		}
	}

	// Добавляем последний label
	if currentLabel != "" && currentValue != "" {
		labels[currentLabel] = currentValue
	}

	return name, labels, nil
}

// FilterMetrics фильтрует метрики по различным критериям
func (p *MetricParser) FilterMetrics(metrics []MetricData, filters map[string]string) []MetricData {
	if len(filters) == 0 {
		return metrics
	}

	var filtered []MetricData
	for _, metric := range metrics {
		if p.matchesFilters(metric, filters) {
			filtered = append(filtered, metric)
		}
	}
	return filtered
}

// matchesFilters проверяет соответствует ли метрика фильтрам
func (p *MetricParser) matchesFilters(metric MetricData, filters map[string]string) bool {
	for key, value := range filters {
		switch key {
		case "name":
			if metric.Name != value {
				return false
			}
		case "type":
			if metric.Type != value {
				return false
			}
		default:
			// Проверяем labels
			if metricValue, exists := metric.Labels[key]; !exists || metricValue != value {
				return false
			}
		}
	}
	return true
}

// GroupMetricsBy группирует метрики по различным критериям
func (p *MetricParser) GroupMetricsBy(metrics []MetricData, groupBy string) map[string][]MetricData {
	groups := make(map[string][]MetricData)

	for _, metric := range metrics {
		var groupKey string

		switch groupBy {
		case "name":
			groupKey = metric.Name
		case "type":
			groupKey = metric.Type
		case "label":
			// Группировка по первому label (пример)
			if len(metric.Labels) > 0 {
				for _, v := range metric.Labels {
					groupKey = v
					break
				}
			} else {
				groupKey = "no_labels"
			}
		default:
			groupKey = "all"
		}

		groups[groupKey] = append(groups[groupKey], metric)
	}

	return groups
}

// PrintMetrics печатает метрики в читаемом формате
func (p *MetricParser) PrintMetrics(metrics []MetricData, showDetails bool) {
	for i, metric := range metrics {
		fmt.Printf("%d. %s", i+1, metric.Name)

		if len(metric.Labels) > 0 {
			fmt.Printf("%s", p.formatLabels(metric.Labels))
		}

		fmt.Printf(" = %f", metric.Value)

		if showDetails {
			fmt.Printf(" [%s]", metric.Type)
			if metric.Help != "" {
				fmt.Printf(" - %s", metric.Help)
			}
			if !metric.Timestamp.IsZero() {
				fmt.Printf(" @ %s", metric.Timestamp.Format(time.RFC3339))
			}
		}

		fmt.Println()
	}
}

// formatLabels форматирует labels для вывода
func (p *MetricParser) formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("{")
	first := true
	for k, v := range labels {
		if !first {
			builder.WriteString(", ")
		}
		builder.WriteString(fmt.Sprintf("%s=%q", k, v))
		first = false
	}
	builder.WriteString("}")
	return builder.String()
}

// GetMetricNames возвращает уникальные имена метрик
func (p *MetricParser) GetMetricNames(metrics []MetricData) []string {
	names := make(map[string]bool)
	for _, metric := range metrics {
		names[metric.Name] = true
	}

	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	return result
}

// GetMetricValues возвращает значения для конкретной метрики
func (p *MetricParser) GetMetricValues(metrics []MetricData, metricName string) []float64 {
	var values []float64
	for _, metric := range metrics {
		if metric.Name == metricName {
			values = append(values, metric.Value)
		}
	}
	return values
}
