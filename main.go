package main

import (
	"context"
	"os/signal"
	"syscall"

	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"chrdsdatamanager/tlsutil"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gocql/gocql"

	"chrdsdatamanager/docs"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

const (
	// Version
	Version string = "1.1.0"
	// LogFile General log file
	LogFile string = "chrdsdatamanager.log"
	// ConfigFile General configuration file
	ConfigFile string = "chrdsdatamanager.json"
)

// FlagLog Type of log FILE - STDOUT
var FlagLog string

var Session *gocql.Session
var ConsistencyRead gocql.Consistency

type ResponseVersionT struct {
	Version    string `json:"version"`
	APIVersion string `json:"apiversion"`
}

type RequestRawDataSaveT struct {
	SpaceID   string  `json:"spaceid"`
	Metric    string  `json:"metric"`
	Value     float32 `json:"value"`
	EventTime int64   `json:"eventtime"`
	Object    string  `json:"object"`
}

type RequestRawDataSaveArrayT struct {
	Data []RequestRawDataSaveT `json:"data"`
}

type RequestRawTextSaveT struct {
	SpaceID   string `json:"spaceid"`
	Metric    string `json:"metric"`
	Value     string `json:"value"`
	EventTime int64  `json:"eventtime"`
	Object    string `json:"object"`
}

type RequestRawTextSaveArrayT struct {
	Data []RequestRawTextSaveT `json:"data"`
}

type ResponseRawDataSaveT struct {
	SpaceID string `json:"spaceid"`
	Metric  string `json:"metric"`
	Object  string `json:"object"`
	Status  int    `json:"status"`
}

type ResponseRawDataSaveArrayT struct {
	Data []ResponseRawDataSaveT `json:"data"`
}

type SMTPMessageT struct {
	ToA     []string
	Subject string
	Body    string
}

var SignalChan chan os.Signal

var SMTPQueue chan SMTPMessageT

func init() {
	flag.StringVar(&FlagLog, "log", "stdout", "[-log file, stdout (default)]")
	flag.Parse()
	if FlagLog == "file" {
		fileLog, err := os.OpenFile(LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0664)
		if err != nil {
			log.Print("An error in opening a log file!")
			os.Exit(-1)
		}
		log.SetOutput(fileLog)
	}

	err := AppConfig.LoadConfig()
	if err != nil {
		log.Fatal("Configuration loading error (", err.Error(), ")!")
		return
	}
}

func done() {
}

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name api_key
func main() {
	defer done()
	defer log.Print("The service is stopped!")

	// Обработка сигналов для graceful shutdown
	SignalChan = make(chan os.Signal, 1)
	signal.Notify(SignalChan, syscall.SIGINT, syscall.SIGTERM)

	// Connect to DB cluster
	cluster := gocql.NewCluster(AppConfig.DB.HOSTS...)
	cluster.Timeout = time.Duration(AppConfig.DB.TIMEOUT) * time.Millisecond
	cluster.Keyspace = AppConfig.DB.KEYSPACE

	AppConfig.DB.CONSISTENCY = strings.ToUpper(AppConfig.DB.CONSISTENCY)
	switch AppConfig.DB.CONSISTENCY {
	case "QUORUM":
		cluster.Consistency = gocql.Quorum
	case "ANY":
		cluster.Consistency = gocql.Any
	case "ONE":
		cluster.Consistency = gocql.One
	case "TWO":
		cluster.Consistency = gocql.Two
	case "THREE":
		cluster.Consistency = gocql.Three
	case "ALL":
		cluster.Consistency = gocql.All
	case "LOCALQUORUM":
		cluster.Consistency = gocql.LocalQuorum
	case "EACHQUORUM":
		cluster.Consistency = gocql.EachQuorum
	case "LOCALONE":
		cluster.Consistency = gocql.LocalOne
	default:
		cluster.Consistency = gocql.Quorum
	}
	AppConfig.DB.CONSISTENCYREAD = strings.ToUpper(AppConfig.DB.CONSISTENCYREAD)
	switch AppConfig.DB.CONSISTENCYREAD {
	case "QUORUM":
		ConsistencyRead = gocql.Quorum
	case "ANY":
		ConsistencyRead = gocql.Any
	case "ONE":
		ConsistencyRead = gocql.One
	case "TWO":
		ConsistencyRead = gocql.Two
	case "THREE":
		ConsistencyRead = gocql.Three
	case "ALL":
		ConsistencyRead = gocql.All
	case "LOCALQUORUM":
		ConsistencyRead = gocql.LocalQuorum
	case "EACHQUORUM":
		ConsistencyRead = gocql.EachQuorum
	case "LOCALONE":
		ConsistencyRead = gocql.LocalOne
	default:
		ConsistencyRead = gocql.Quorum
	}

	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: AppConfig.DB.USERNAME,
		Password: AppConfig.DB.PASSWORD,
	}
	if AppConfig.DB.TLS {
		crt, err := tlsutil.LoadX509KeyPairWithPassword(AppConfig.TLS.CERTPATH, AppConfig.TLS.KEYPATH, []byte(AppConfig.TLS.KEYPASSWORD))
		if err != nil {
			log.Fatal("Failed to download the certificate (", err, ")!")
			return
		}
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

		cluster.SslOpts = &gocql.SslOptions{
			Config: &tls.Config{
				RootCAs:            crtPool,
				Certificates:       []tls.Certificate{crt},
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: AppConfig.DB.SKIPVERIFY,
			},
		}
	}

	var err error
	Session, err = cluster.CreateSession()
	if err != nil {
		log.Fatal("It was not possible to establish a connection with the database (", err, ")!")
	} else {
		log.Print("The connection to the database is established... OK")
	}
	defer Session.Close()

	// ZBX Server start
	go zbxServer()

	// WorkerPool ZBX start
	go zbxWorkersRun()

	// SMTP QUEUE init
	if AppConfig.SMTP.ENABLED {
		queueSize := 100 // Default value QUEUE
		if AppConfig.SMTP.QUEUE > 0 {
			queueSize = AppConfig.SMTP.QUEUE
		}
		SMTPQueue = make(chan SMTPMessageT, queueSize)
		defer close(SMTPQueue)
		// SMTP Consumer start
		go smtpConsumer()
	}

	// WorkerPool Prometheus scraper start
	if AppConfig.PRMTSCRAPER.ENABLED {
		scraper := NewScraper() // Создаем скрапер
		scraper.workersStart()
		defer scraper.Stop()
	}

	go httpServer()

	// Ожидаем сигнал остановки
	<-SignalChan
	log.Print("Received shutdown signal, exiting...")

	close(SignalChan)
}

func httpServer() {
	// HTTP Server
	docs.SwaggerInfo.Title = AppConfig.SWG.TITLE
	docs.SwaggerInfo.Description = AppConfig.SWG.DESCRIPTION
	docs.SwaggerInfo.Version = AppConfig.SWG.VERSION
	docs.SwaggerInfo.Host = AppConfig.SWG.HOST
	docs.SwaggerInfo.BasePath = AppConfig.SWG.BASEPATH
	docs.SwaggerInfo.Schemes = []string{AppConfig.SWG.SCHEME}

	r := chi.NewRouter() // Initialization GoChi Router

	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger) // Expanded logging
	r.Use(middleware.URLFormat)

	r.Use(httprate.Limit(
		AppConfig.HTTP.RL, // Requests
		1*time.Second,     // Per duration
		httprate.WithKeyFuncs(func(r *http.Request) (string, error) {
			return r.Header.Get("api_key"), nil
		}),
	))

	// Requests to the API
	r.Group(func(r chi.Router) {
		r.Route("/api/v{version}/version", func(r chi.Router) {
			r.Get("/", getVersion)
		})
		r.Route("/api/v{version}/data", func(r chi.Router) {
			r.Post("/save", postRawDataSave)
		})
		r.Route("/api/v{version}/text", func(r chi.Router) {
			r.Post("/save", postRawTextSave)
		})
	})

	r.Mount("/swagger", httpSwagger.WrapHandler) // Swagger UI
	if AppConfig.PRMTEXPORTER.ENABLED {
		r.Mount(AppConfig.PRMTEXPORTER.METRICSPATH, promhttp.Handler()) // Prometheus metrics
	}

	if AppConfig.HTTP.TLS { // HTTPS connection
		// crt, err := tls.LoadX509KeyPair(AppConfig.TLS.CERTPATH, AppConfig.TLS.KEYPATH)
		crt, err := tlsutil.LoadX509KeyPairWithPassword(AppConfig.TLS.CERTPATH, AppConfig.TLS.KEYPATH, []byte(AppConfig.TLS.KEYPASSWORD))
		if err != nil {
			log.Fatal("Failed to download the certificate (", err, ")!")
			return
		}

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
		tlsConfig := &tls.Config{
			RootCAs:            crtPool,
			Certificates:       []tls.Certificate{crt},
			InsecureSkipVerify: AppConfig.HTTP.SKIPVERIFY,
			MinVersion:         tls.VersionTLS12,
		}

		httpServer := &http.Server{Addr: AppConfig.HTTP.HOST + ":" + AppConfig.HTTP.PORT,
			Handler:      r,
			ReadTimeout:  time.Duration(AppConfig.HTTP.READTIMEOUT) * time.Second,
			WriteTimeout: time.Duration(AppConfig.HTTP.WRITETIMEOUT) * time.Second,
			TLSConfig:    tlsConfig}
		log.Print("Datamanager service is launched at the port ", AppConfig.HTTP.PORT, "(HTTPS)... ОК")
		httpServer.ListenAndServeTLS("", "")
	} else { // HTTP соединение
		httpServer := &http.Server{Addr: AppConfig.HTTP.HOST + ":" + AppConfig.HTTP.PORT,
			Handler:      r,
			ReadTimeout:  time.Duration(AppConfig.HTTP.READTIMEOUT) * time.Second,
			WriteTimeout: time.Duration(AppConfig.HTTP.WRITETIMEOUT) * time.Second}
		log.Print("Datamanager service is launched at the port ", AppConfig.HTTP.PORT, "(HTTP)... ОК")
		httpServer.ListenAndServe()
	}
}

// @Summary		Get the version of the DataManager module
// @Tags		Version
// @Accept		json
// @Produce		json
// @Success		200 {object} ResponseVersionT
// @Failure     403 "Forbidden, The monitoring module key is incorrect, or not registered"
// @Failure     500 "Internal server error, The internal error of the service"
// @Router		/version [get]
// @Security	ApiKeyAuth
func getVersion(w http.ResponseWriter, r *http.Request) {
	versionAPI := chi.URLParam(r, "version")

	if registrationCheck(r.Header.Get("api_key")) {
		if versionAPI == "1" {
			responseVersion := &ResponseVersionT{
				Version:    Version,
				APIVersion: versionAPI,
			}

			responseVersionJSON, err := json.Marshal(responseVersion)
			if err != nil {
				log.Print("JSON MARSHAL ERROR (" + err.Error() + ")!")
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.Write(responseVersionJSON)
			}
		}
	} else {
		w.WriteHeader(http.StatusForbidden)
	}
}

// @Summary		Save monitoring data in the database
// @Tags		Data
// @Accept		json
// @Produce		json
// @Param		data body RequestRawDataSaveArrayT true "Input structure"
// @Success		200 {object} ResponseRawDataSaveArrayT "OK, The status of 200 is always returned, the data is successfully preserved if in the return structure status = 1"
// @Failure     403 "Forbidden, The monitoring module key is incorrect, or not registered"
// @Failure     500 "Internal server error, The internal error of the service"
// @Router		/data/save [post]
// @Security	ApiKeyAuth
func postRawDataSave(w http.ResponseWriter, r *http.Request) {
	versionAPI := chi.URLParam(r, "version")
	moduleID := r.Header.Get("api_key")

	ctx := context.Background()
	if registrationCheck(moduleID) {
		if versionAPI == "1" {
			b, err := io.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var requestRawDataSaveArray RequestRawDataSaveArrayT
			var responseRawDataSaveArray ResponseRawDataSaveArrayT

			if err := json.Unmarshal(b, &requestRawDataSaveArray); err != nil {
				log.Print("JSON UNMARSHAL ERROR (" + err.Error() + ")!")
				w.WriteHeader(http.StatusInternalServerError)
				return
			} else {
				for index := range requestRawDataSaveArray.Data {
					var responseRawDataSave ResponseRawDataSaveT
					responseRawDataSave.SpaceID = requestRawDataSaveArray.Data[index].SpaceID
					responseRawDataSave.Metric = requestRawDataSaveArray.Data[index].Metric

					spaceStatus := 0
					spaceDescription := ""
					if err := Session.Query(`SELECT status, description FROM space WHERE id = ? LIMIT 1`, requestRawDataSaveArray.Data[index].SpaceID).WithContext(ctx).Consistency(ConsistencyRead).Scan(&spaceStatus, &spaceDescription); err != nil {
						log.Print(err)
					}
					if spaceStatus == 1 {
						log.Print("The raw_data table is used...")
						err := Session.Query(`
							BEGIN BATCH
								INSERT INTO raw_data (metric, value, create_time, event_time, space_id, object, space_description) VALUES (?, ?, toTimestamp(now()), ?, ?, ?, ?)
								INSERT INTO space_metric_data (space_id, metric) VALUES (?, ?)
							APPLY BATCH
							`,
							requestRawDataSaveArray.Data[index].Metric, requestRawDataSaveArray.Data[index].Value,
							requestRawDataSaveArray.Data[index].EventTime, requestRawDataSaveArray.Data[index].SpaceID,
							requestRawDataSaveArray.Data[index].Object, spaceDescription,
							requestRawDataSaveArray.Data[index].SpaceID, requestRawDataSaveArray.Data[index].Metric).WithContext(ctx).Exec()
						if err != nil {
							log.Print(err)
							responseRawDataSave.Status = 0
						} else {
							go problem(requestRawDataSaveArray.Data[index].SpaceID, moduleID, requestRawDataSaveArray.Data[index].Metric, makeTimestamp(), requestRawDataSaveArray.Data[index].Value, "", "api") // Problem check
							responseRawDataSave.Status = 1
						}
					} else {
						log.Printf("The space of %v is blocked, the recording is impossible!", requestRawDataSaveArray.Data[index].SpaceID)
						responseRawDataSave.Status = 0
					}
					responseRawDataSaveArray.Data = append(responseRawDataSaveArray.Data, responseRawDataSave)
				}

				responseRawDataSaveJSON, err := json.Marshal(responseRawDataSaveArray)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				} else {
					w.Header().Set("Content-Type", "application/json")
					w.Write(responseRawDataSaveJSON)
				}
				w.WriteHeader(http.StatusOK)
			}
		}
	} else {
		w.WriteHeader(http.StatusForbidden)
	}
}

// @Summary		Save text monitoring data in the database
// @Tags		Data
// @Accept		json
// @Produce		json
// @Param		data body RequestRawTextSaveArrayT true "Input structure"
// @Success		200 {object} ResponseRawDataSaveArrayT "OK, The status of 200 is always returned, the data is successfully preserved if in the return structure status = 1"
// @Failure     403 "Forbidden, The monitoring module key is incorrect, or not registered"
// @Failure     500 "Internal server error, The internal error of the service"
// @Router		/text/save [post]
// @Security	ApiKeyAuth
func postRawTextSave(w http.ResponseWriter, r *http.Request) {
	versionAPI := chi.URLParam(r, "version")

	ctx := context.Background()
	if registrationCheck(r.Header.Get("api_key")) {
		if versionAPI == "1" {
			b, err := io.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var requestRawTextSaveArray RequestRawTextSaveArrayT
			var responseRawDataSaveArray ResponseRawDataSaveArrayT

			if err := json.Unmarshal(b, &requestRawTextSaveArray); err != nil {
				log.Print("JSON UNMARSHAL ERROR (" + err.Error() + ")!")
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				for index := range requestRawTextSaveArray.Data {
					var responseRawDataSave ResponseRawDataSaveT
					responseRawDataSave.SpaceID = requestRawTextSaveArray.Data[index].SpaceID
					responseRawDataSave.Metric = requestRawTextSaveArray.Data[index].Metric

					spaceStatus := 0
					spaceDescription := ""
					if err := Session.Query(`SELECT status, description FROM space WHERE id = ? LIMIT 1`, requestRawTextSaveArray.Data[index].SpaceID).WithContext(ctx).Consistency(ConsistencyRead).Scan(&spaceStatus, &spaceDescription); err != nil {
						log.Print(err)
					}
					if spaceStatus == 1 {

						log.Print("The raw_text table is used...")
						err := Session.Query(`
							BEGIN BATCH
								INSERT INTO raw_text (metric, value, create_time, event_time, space_id, object, space_description) VALUES (?, ?, toTimestamp(now()), ?, ?, ?, ?)
								INSERT INTO space_metric_text (space_id, metric) VALUES (?, ?)
							APPLY BATCH`,
							requestRawTextSaveArray.Data[index].Metric, requestRawTextSaveArray.Data[index].Value,
							requestRawTextSaveArray.Data[index].EventTime, requestRawTextSaveArray.Data[index].SpaceID,
							requestRawTextSaveArray.Data[index].Object, spaceDescription,
							requestRawTextSaveArray.Data[index].SpaceID, requestRawTextSaveArray.Data[index].Metric).WithContext(ctx).Exec()
						if err != nil {
							log.Print(err)
							responseRawDataSave.Status = 0
						} else {
							responseRawDataSave.Status = 1
							spaceIDCount, err := gocql.ParseUUID(requestRawTextSaveArray.Data[index].SpaceID)
							if err != nil {
								log.Print(err)
							} else {
								err := rawCount(spaceIDCount, "text", 1)
								if err != nil {
									log.Print(err)
								}
							}
							log.Print("TEXT SAVE")
						}
					} else {
						log.Printf("The space of %v is blocked, the recording is impossible!", requestRawTextSaveArray.Data[index].SpaceID)
						responseRawDataSave.Status = 0
					}
					responseRawDataSaveArray.Data = append(responseRawDataSaveArray.Data, responseRawDataSave)
				}

				responseRawDataSaveJSON, err := json.Marshal(responseRawDataSaveArray)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
				} else {
					w.Header().Set("Content-Type", "application/json")
					w.Write(responseRawDataSaveJSON)
				}
				w.WriteHeader(http.StatusOK)
			}
		}
	} else {
		w.WriteHeader(http.StatusForbidden)
	}
}

func registrationCheck(id string) bool {
	ctx := context.Background()
	scanner := Session.Query(`SELECT status FROM registration WHERE id = ?`, id).WithContext(ctx).Consistency(ConsistencyRead).Iter().Scanner()
	for scanner.Next() {
		var status int

		err := scanner.Scan(&status)
		if err != nil {
			return false
		} else {
			if status == 1 {
				return true
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Print(err)
		return false
	}
	return false
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func tlsVersionToString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return "Unknown"
	}
}

func problem(spaceID string, modulID string, metric string, eventTime int64, valueF float32, valueS string, object string) {
	ctx := context.Background()

	scanner01 := Session.Query(`SELECT critical, warning, data_type, emaillist_id FROM task WHERE module_id = ? AND space_id = ? AND object = ? AND metric = ?`, modulID, spaceID, object, metric).WithContext(ctx).Consistency(ConsistencyRead).Iter().Scanner()
	for scanner01.Next() {
		var critical string
		var warning string
		var dataType string
		var emailListID gocql.UUID

		var emailA []string

		err := scanner01.Scan(&critical, &warning, &dataType, &emailListID)
		if err != nil {
			log.Print(err)
			return
		} else {
			if emailListID != (gocql.UUID{}) {
				scannerEMail := Session.Query(`SELECT email, status FROM emaillist_data WHERE emaillist_id = ?`, emailListID.String()).WithContext(ctx).Consistency(ConsistencyRead).Iter().Scanner()
				for scannerEMail.Next() {
					var email string
					var status int

					err := scannerEMail.Scan(&email, &status)
					if err != nil {
						log.Print(err)
					} else {
						if status == 1 {
							emailA = append(emailA, email)
						}
					}
				}
			}

			status := "NORMAL"
			var valueAllType string
			if dataType == "float" {
				valueAllType = strconv.FormatFloat(float64(valueF), 'f', -1, 32)

				if critical != "" && warning != "" {
					criticalFloat, err01 := strconv.ParseFloat(critical, 32)
					if err01 != nil {
						log.Print(err01)
						return
					}
					warningFloat, err02 := strconv.ParseFloat(warning, 32)
					if err02 != nil {
						log.Print(err02)
					}
					if criticalFloat >= warningFloat {
						if valueF >= float32(criticalFloat) {
							log.Print("CRITICAL")
							status = "CRITICAL"
						} else {
							if valueF >= float32(warningFloat) {
								log.Print("WARNING")
								status = "WARNING"
							}
						}
					} else {
						if valueF <= float32(criticalFloat) {
							log.Print("CRITICAL")
							status = "CRITICAL"
						} else {
							if valueF <= float32(warningFloat) {
								log.Print("WARNING")
								status = "WARNING"
							}
						}
					}
				} else if critical != "" && warning == "" {
					criticalFloat, err01 := strconv.ParseFloat(critical, 32)
					if err01 != nil {
						log.Print(err01)
						return
					}
					if valueF == float32(criticalFloat) {
						log.Print("CRITICAL")
						status = "CRITICAL"
					}
				} else if critical == "" && warning != "" {
					warningFloat, err02 := strconv.ParseFloat(warning, 32)
					if err02 != nil {
						log.Print(err02)
						return
					}
					if valueF == float32(warningFloat) {
						log.Print("WARNING")
						status = "WARNING"
					}
				}
			} else {
				valueAllType = valueS
			}
			scanner02 := Session.Query(`SELECT status, event_time_start FROM problem WHERE module_id = ? AND space_id = ? AND metric = ?`, modulID, spaceID, metric).WithContext(ctx).Consistency(ConsistencyRead).Iter().Scanner()
			countProblem := 0
			for scanner02.Next() {
				var statusDB string
				var eventTimeStartDB int64
				err := scanner02.Scan(&statusDB, &eventTimeStartDB)
				if err != nil {
					log.Print(err)
					return
				} else {
					if status != statusDB {
						applied, err := Session.Query(`UPDATE problem SET event_time = ?, status = ?, value = ? WHERE space_ID = ? AND module_id = ? AND metric = ? IF status != ?
						`, eventTime, status, valueAllType, spaceID, modulID, metric, status).WithContext(ctx).ScanCAS(nil)
						if err != nil {
							log.Print(err)
							return
						} else {
							if applied {
								log.Printf("Optimistic block problem update... OK")
								log.Print("PROBLEM UPDATE")
								if len(emailA) > 0 {
									go smtpProducer(emailA, status, ("CHRDS" + " " + status + " " + spaceName(spaceID) + "(" + moduleName(modulID) + ")"), (metric + " " + valueAllType + " " + statusDB + " -> " + status))
								}
							} else {
								log.Printf("Optimistic block problem update... FALSE")
							}
						}
					} else {
						log.Print("Problem time: ", eventTime-eventTimeStartDB)
						if statusDB == "NORMAL" && (eventTime-eventTimeStartDB) > 3600*1000 { // Добавить параметризацию времени хранения статуса восстановления (NORMAL)
							applied, err := Session.Query(`DELETE FROM problem WHERE space_ID = ? AND module_id = ? AND metric = ? IF EXISTS`, spaceID, modulID, metric).WithContext(ctx).ScanCAS(nil)
							if err != nil {
								log.Print(err)
								return
							} else {
								if applied {
									log.Printf("Optimistic block problem delete... OK")
									log.Print("PROBLEM DELETE (NORMAL)")
								} else {
									log.Printf("Optimistic block problem delete... FALSE")
								}
							}
						} else {
							applied, err := Session.Query(`UPDATE problem SET event_time = ?, value = ? WHERE space_ID = ? AND module_id = ? AND metric = ?  IF status != ?`, eventTime, valueAllType, spaceID, modulID, metric, status).WithContext(ctx).ScanCAS(nil)
							if err != nil {
								log.Print(err)
								return
							} else {
								if applied {
									log.Printf("Optimistic block problem update... OK")
									log.Print("PROBLEM UPDATE")
								} else {
									log.Printf("Optimistic block problem update... FALSE")
								}
							}
						}
					}
				}
				countProblem++
			}
			if countProblem == 0 && status != "NORMAL" {
				applied, err := Session.Query(`INSERT INTO problem (space_id, module_id, metric, event_time, event_time_start, status, value) VALUES (?, ?, ?, ?, ?, ?, ?) IF NOT EXISTS
				`, spaceID, modulID, metric, eventTime, eventTime, status, valueAllType).WithContext(ctx).ScanCAS(nil)
				if err != nil {
					log.Print(err)
					return
				} else {
					if applied {
						log.Printf("Optimistic block problem insert... OK")
						log.Print("PROBLEM INSERT")
						if len(emailA) > 0 {
							go smtpProducer(emailA, status, ("CHRDS" + " " + status + " " + spaceName(spaceID) + "(" + moduleName(modulID) + ")"), (metric + " " + valueAllType + " " + status))
						}
					} else {
						log.Printf("Optimistic block problem insert... FALSE")
						log.Print("Problem already exists!")
					}
				}
			}
		}
	}
}

func spaceName(spaceID string) string {
	ctx := context.Background()
	var spaceName string
	if err := Session.Query(`SELECT description FROM space WHERE id = ? LIMIT 1`, spaceID).WithContext(ctx).Consistency(ConsistencyRead).Scan(&spaceName); err != nil {
		log.Print(err)
		return ""
	}
	return spaceName
}

func moduleName(moduleID string) string {
	ctx := context.Background()
	var moduleName string
	if err := Session.Query(`SELECT description FROM registration WHERE id = ? LIMIT 1`, moduleID).WithContext(ctx).Consistency(ConsistencyRead).Scan(&moduleName); err != nil {
		log.Print(err)
		return ""
	}
	return moduleName
}
