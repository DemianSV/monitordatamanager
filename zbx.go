package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"monitordatamanager/tlsutil"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocql/gocql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const ZBXHeaderSize = 4 + 1 + 4 + 4
const ZBXTCPProtocol = byte(0x01)
const ZBXZLibCompress = byte(0x03)
const ZBXCompress = true

var ZBXAgentMap map[string]string

var (
	zbxClientDuration = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "zbxclient_request_duration_seconds",
			Help: "Duration of ZBXClient HTTP requests in seconds",
		},
		[]string{"target"},
	)

	zbxClientDurationFull = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "zbxclent_request_duration_full_seconds",
			Help: "Duration of ZBXClient HTTP requests + INSERT DB in seconds",
		},
		[]string{"target"},
	)

	activeZBXWorkers = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "zbx_active_workers",
			Help: "Number of active ZBX Client workers",
		},
		[]string{},
	)
)

type RequestAgentTypeT struct {
	Request string `json:"request"`
}

type RequestAgentT struct {
	Request string `json:"request"`
	Host    string `json:"host"`
	Version string `json:"version"`
	Session string `json:"session"`
}

type RequestAgentMetricsDataT struct {
	ID     int32  `json:"id"`
	ItemID int64  `json:"itemid"`
	Value  string `json:"value"`
	Clock  int32  `json:"clock"`
	NS     int32  `json:"ns"`
}

type RequestAgentMetricsT struct {
	Request string                     `json:"request"`
	Host    string                     `json:"host"`
	Data    []RequestAgentMetricsDataT `json:"data"`
	Session string                     `json:"session"`
}

var AgentStart map[string]bool

type ResponseServerDataT struct {
	Itemid      int64  `json:"itemid"`
	Key         string `json:"key"`
	KeyOrig     string `json:"key_orig"`
	Delay       string `json:"delay"`
	Lastlogsize int    `json:"lastlogsize"`
	Mtime       int    `json:"mtime"`
}

type ResponseServerT struct {
	Response string                `json:"response"`
	Data     []ResponseServerDataT `json:"data"`
}

type ResponsServerMetricsT struct {
	Response string `json:"response"`
	Info     string `json:"info"` // "info": "processed: 2; failed: 0; total: 2; seconds spent: 0.003534"
}

type JobT struct {
	key   string
	value string
}

func zbxServer() {
	zbxType := AppConfig.ZBX.TYPE
	zbxURL := AppConfig.ZBX.HOST + ":" + AppConfig.ZBX.PORT

	AgentStart = make(map[string]bool)
	ZBXAgentMap = make(map[string]string)
	var mutex01 sync.Mutex
	var lsn net.Listener

	// TLS for ZBX server
	if AppConfig.ZBX.TLS {
		caCert, err := os.ReadFile(AppConfig.TLS.CAPATH)
		if err != nil {
			log.Fatal("Failed to open CA Certificate!")
		}
		ca := x509.NewCertPool()
		statusCA := ca.AppendCertsFromPEM(caCert)
		if !statusCA {
			panic("ZBX Server: Failed to parse CA certificate!")
		}

		crt, err := tlsutil.LoadX509KeyPairWithPassword(AppConfig.TLS.CERTPATH, AppConfig.TLS.KEYPATH, []byte(AppConfig.TLS.KEYPASSWORD))
		if err != nil {
			log.Fatal("Failed to download the certificate (", err, ")!")
			return
		}

		config := tls.Config{
			Certificates:       []tls.Certificate{crt},
			ClientAuth:         tls.RequireAnyClientCert,
			InsecureSkipVerify: AppConfig.ZBX.SKIPVERIFY,
			RootCAs:            ca,
			MinVersion:         tls.VersionTLS12, // Minimum version TLS 1.2
		}

		lsn, err = tls.Listen(zbxType, zbxURL, &config)
		if err != nil {
			log.Print("Error listening ZBX: ", err.Error())
			return
		}
	} else {
		var err error
		lsn, err = net.Listen(zbxType, zbxURL)
		if err != nil {
			log.Print("Error listening ZBX: ", err.Error())
			return
		}
	}

	// Close the listener when this application closes
	defer lsn.Close()

	log.Print("ZBX service is launched and available on " + zbxURL + "... OK")
	for {
		conn, err := lsn.Accept()
		if err != nil {
			log.Print("Error ZBX accepting connection: ", err.Error())
			continue
		}

		go zbxHandleRequest(conn, &mutex01)
	}
}

func zbxHandleRequest(conn net.Conn, mutex01 *sync.Mutex) {
	defer conn.Close()
	ctx := context.Background()

	if AppConfig.ZBX.TLS {
		if connTLS, ok := conn.(*tls.Conn); ok {
			err := connTLS.Handshake()
			if err != nil {
				log.Printf("ZBX Server: No connection established, ERROR TLS HANDSHAKE (%v)", err)
				return
			}

			state := connTLS.ConnectionState()
			log.Printf("ZBX Server: TLS connection is established. TLS Version: %s, TLS ChipherSuite: %v, TLS Handshake: %v\n",
				tlsVersionToString(state.Version),
				state.CipherSuite,
				state.HandshakeComplete)
		}
	}

	var requestAgentType RequestAgentTypeT
	var requestAgent RequestAgentT
	var responseServerData ResponseServerDataT
	var responseServerDataSlice []ResponseServerDataT
	var responseServer ResponseServerT
	var requestAgentMetrics RequestAgentMetricsT
	var responseServerMetrics ResponsServerMetricsT

	hdr := make([]byte, 13)
	tmp := make([]byte, 1024)
	buf := make([]byte, 0)

	log.Print(strings.Split(conn.RemoteAddr().String(), ":")[0])

	sizeRead, err := conn.Read(hdr)
	if err != nil {
		log.Print(err)
	} else {
		log.Print("ZBX Header read: ", sizeRead)

		protocolFlag := int(hdr[4])
		log.Print("ZBX Protocol FLAG: ", protocolFlag)
		protocolSize := (hdr[5:12])
		var protocolSizeInt uint32
		hdr01 := bytes.NewReader(protocolSize)
		err = binary.Read(hdr01, binary.LittleEndian, &protocolSizeInt)
		if err != nil {
			log.Print("Binary read failed: ", err)
		} else {
			buf = append(buf, hdr[:13]...)

			var dataSize uint32
			dataSize = 0
			for {
				if dataSize == protocolSizeInt+13 {
					break
				}
				sizeRead, err := conn.Read(tmp)
				if err != nil {
					log.Print("Error reading: ", err.Error())
					break
				}
				log.Print("Read: ", sizeRead)
				buf = append(buf, tmp[:sizeRead]...)
				dataSize = uint32(len(buf))
			}

			sizeRead := len(buf)
			log.Print("END Read: ", sizeRead)
			protocolData := (buf[13 : 13+protocolSizeInt])

			uncompressData, err := zbxUncompress(protocolData, protocolSizeInt, protocolFlag)
			if err != err {
			} else {
				if err := json.Unmarshal(uncompressData, &requestAgentType); err != nil {
					log.Print(err)
				} else {
					log.Print(requestAgentType.Request)
					switch requestAgentType.Request {
					case "active checks":
						if err := json.Unmarshal(uncompressData, &requestAgent); err != nil {
							log.Print(err)
						} else {
							if registrationCheck(requestAgent.Host) {
								log.Print(requestAgent)

								scanner := Session.Query(`SELECT object, metric, status, int_id, interval FROM task WHERE module_id = ?`, requestAgent.Host).WithContext(ctx).Consistency(ConsistencyRead).Iter().Scanner()
								for scanner.Next() {
									var object string
									var metric string
									var status int
									var intID int64
									var interval int64
									err := scanner.Scan(&object, &metric, &status, &intID, &interval)
									if err != nil {
										log.Print(err)
										return
									} else {
										if interval == 0 {
											interval = 10
										}
										if status == 1 && object == "zbxagent" {
											responseServerData.Itemid = intID
											responseServerData.Key = metric
											responseServerData.KeyOrig = metric
											responseServerData.Delay = fmt.Sprintf("%ds", interval)
											responseServerData.Lastlogsize = 0
											responseServerData.Lastlogsize = 0
											responseServerDataSlice = append(responseServerDataSlice, responseServerData)
										}
									}
								}
								responseServer.Response = "success"
								responseServer.Data = responseServerDataSlice

								responseServerJSON, err := json.Marshal(responseServer)
								if err != nil {
									log.Print(err)
								} else {
									log.Print(string(responseServerJSON))

									var buf02 bytes.Buffer
									flags := ZBXTCPProtocol
									if ZBXCompress {
										z := zlib.NewWriter(&buf02)
										if _, err = z.Write(responseServerJSON); err != nil {
											log.Print(err)
										}
										z.Close()
										flags |= ZBXZLibCompress
									} else {
										buf02.Write(responseServerJSON)
									}

									var buf03 bytes.Buffer
									buf03.Grow(buf02.Len() + ZBXHeaderSize)
									buf03.Write([]byte{'Z', 'B', 'X', 'D', flags})
									if err = binary.Write(&buf03, binary.LittleEndian, uint32(buf02.Len())); nil != err {
										log.Print(err)
									}
									if err = binary.Write(&buf03, binary.LittleEndian, uint32(len(responseServerJSON))); nil != err {
										log.Print(err)
									}
									buf03.Write(buf02.Bytes())
									sizeWrite, err := conn.Write(buf03.Bytes())
									if err != nil {
										log.Print(err)
									} else {
										log.Print(requestAgent.Host)
										AgentStart[requestAgent.Host] = true
										log.Print("Write: ", sizeWrite)
									}
								}
							} else {
								log.Print("Request from an unregistered agent!")
							}
						}
					case "agent data":
						if err := json.Unmarshal(uncompressData, &requestAgentMetrics); err != nil {
							log.Print(err)
						} else {
							if registrationCheck(requestAgentMetrics.Host) {
								if AgentStart[requestAgentMetrics.Host] {
									log.Print(requestAgentMetrics)
									for _, value := range requestAgentMetrics.Data {
										spaceIDDB := ""
										metricDB := ""
										dataTypeDB := ""
										scanner := Session.Query(`SELECT space_id, status, metric, object, int_id, data_type FROM task WHERE module_id = ?`, requestAgentMetrics.Host).WithContext(ctx).Consistency(ConsistencyRead).Iter().Scanner()
										for scanner.Next() {
											var spaceID string
											var status int
											var metric string
											var object string
											var intID int64
											var dataType string
											err := scanner.Scan(&spaceID, &status, &metric, &object, &intID, &dataType)
											if err != nil {
												log.Print(err)
											} else {
												if status == 1 && object == "zbxagent" && intID == int64(value.ItemID) {
													spaceIDDB = spaceID
													metricDB = metric
													dataTypeDB = dataType
												}
											}
										}

										if spaceIDDB != "" && metricDB != "" {
											if dataTypeDB == "float" {
												valueFloat, _ := strconv.ParseFloat(value.Value, 64)
												eventTimeFloat := (float64(value.Clock) * 1000) + (float64(value.NS) / 1000000)
												eventTime := int(math.Round(eventTimeFloat))

												spaceStatus := 0
												spaceDescription := ""
												if err := Session.Query(`SELECT status, description FROM space WHERE id = ? LIMIT 1`, spaceIDDB).WithContext(ctx).Consistency(ConsistencyRead).Scan(&spaceStatus, &spaceDescription); err != nil {
													log.Print(err)
												}
												if spaceStatus == 1 {
													log.Print("The raw_data table is used...")
													err := Session.Query(`
													BEGIN BATCH
														INSERT INTO raw_data (metric, value, create_time, event_time, space_id, object, space_description) VALUES (?, ?, toTimestamp(now()), ?, ?, 'zbxagent', ?)
														INSERT INTO space_metric_data (space_id, metric) VALUES (?, ?)
													APPLY BATCH
													`,
														metricDB, float32(valueFloat),
														eventTime, spaceIDDB, spaceDescription,
														spaceIDDB, metricDB).WithContext(ctx).Exec()
													if err != nil {
														log.Print(err)
													} else {
														spaceID, err := gocql.ParseUUID(spaceIDDB)
														if err != nil {
															log.Print(err)
														} else {
															err = rawCount(spaceID, "data", 1)
															if err != nil {
																log.Print(err)
															}
														}

														log.Print("DATA SAVE")
														mutex01.Lock()
														ZBXAgentMap[strings.Split(conn.RemoteAddr().String(), ":")[0]] = requestAgentMetrics.Host
														mutex01.Unlock()
														go problem(spaceIDDB, requestAgentMetrics.Host, metricDB, int64(eventTime), float32(valueFloat), "", "zbxagent")
													}
												}
											} else {
												eventTime, _ := strconv.ParseInt(fmt.Sprintf("%d%s", value.Clock, "000"), 10, 64)

												spaceStatus := 0
												spaceDescription := ""
												if err := Session.Query(`SELECT status, description FROM space WHERE id = ? LIMIT 1`, spaceIDDB).WithContext(ctx).Consistency(ConsistencyRead).Scan(&spaceStatus, &spaceDescription); err != nil {
													log.Print(err)
												}
												if spaceStatus == 1 {
													log.Print("The raw_text table is used...")
													err := Session.Query(`
														BEGIN BATCH
															INSERT INTO raw_text (metric, value, create_time, event_time, space_id, object, space_description) VALUES (?, ?, toTimestamp(now()), ?, ?, 'zbxagent', ?)
															INSERT INTO space_metric_text (space_id, metric) VALUES (?, ?)
														APPLY BATCH
														`,
														metricDB, value.Value,
														eventTime, spaceIDDB, spaceDescription,
														spaceIDDB, metricDB).WithContext(ctx).Exec()
													if err != nil {
														log.Print(err)
													} else {
														spaceIDCount, err := gocql.ParseUUID(spaceIDDB)
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
												}
											}
										}
									}

									responseServerMetrics.Response = "success"
									responseServerMetrics.Info = fmt.Sprintf("processed: %d; failed: 0; total: %d", len(requestAgentMetrics.Data), len(requestAgentMetrics.Data))
									responseServerJSON, err := json.Marshal(responseServerMetrics)
									log.Print(string(responseServerJSON))
									if err != nil {
										log.Print(err)
									} else {
										var buf04 bytes.Buffer
										flags := ZBXTCPProtocol
										if ZBXCompress {
											z := zlib.NewWriter(&buf04)
											if _, err = z.Write(responseServerJSON); err != nil {
												log.Print(err)
											}
											z.Close()
											flags |= ZBXZLibCompress
										} else {
											buf04.Write(responseServerJSON)
										}

										var buf05 bytes.Buffer
										buf05.Grow(buf04.Len() + ZBXHeaderSize)
										buf05.Write([]byte{'Z', 'B', 'X', 'D', flags})
										if err = binary.Write(&buf05, binary.LittleEndian, uint32(buf04.Len())); nil != err {
											log.Print(err)
										}
										if err = binary.Write(&buf05, binary.LittleEndian, uint32(len(responseServerJSON))); nil != err {
											log.Print(err)
										}
										buf05.Write(buf04.Bytes())
										sizeWrite, err := conn.Write(buf05.Bytes())
										if err != nil {
											log.Print(err)
										} else {
											log.Print("Write: ", sizeWrite)
										}
									}
								}
							} else {
								log.Print("Request from an unregistered agent")
							}
						}
					}
				}
			}
		}
	}
}

func zbxUncompress(protocolData []byte, protocolSizeInt uint32, protocolFlag int) ([]byte, error) {
	if protocolFlag == 3 || protocolFlag == 2 {
		var b bytes.Buffer

		b.Grow(int(protocolSizeInt))
		z, err := zlib.NewReader(bytes.NewReader(protocolData))
		if err != nil {
			log.Printf("Unable to uncompress message (1): '%s'\n", err)
			return nil, err
		}
		len, err := b.ReadFrom(z)
		z.Close()
		if err != nil {
			log.Printf("Unable to uncompress message (2): '%s'\n", err)
			return nil, err
		}
		if len != int64(protocolSizeInt) {
			log.Printf("Uncompressed message size %d instead of expected %d\n", len, protocolSizeInt)
		}
		return b.Bytes(), nil
	} else {
		return protocolData, nil
	}
}

func zbxClient(worker int, jobs <-chan JobT) {
	log.Print("START ZBX WORKER ", worker, "... OK")

	activeZBXWorkers.WithLabelValues().Inc()
	defer activeZBXWorkers.WithLabelValues().Dec()
	defer log.Print("STOP ZBX WORKER ", worker, "... OK")

	var configTLS *tls.Config
	if AppConfig.ZBX.TLS {
		caCert, err := os.ReadFile(AppConfig.TLS.CAPATH)
		if err != nil {
			log.Fatal("Failed to open CA Certificate!")
		}
		ca := x509.NewCertPool()
		statusCA := ca.AppendCertsFromPEM(caCert)
		if !statusCA {
			panic("ZBX Server: Failed to parse CA certificate!")
		}
		crt, err := tlsutil.LoadX509KeyPairWithPassword(AppConfig.TLS.CERTPATH, AppConfig.TLS.KEYPATH, []byte(AppConfig.TLS.KEYPASSWORD))
		if err != nil {
			log.Fatal("Failed to download the certificate (", err, ")!")
			return
		}
		configTLS = &tls.Config{
			Certificates:       []tls.Certificate{crt},
			InsecureSkipVerify: AppConfig.ZBX.SKIPVERIFY,
			RootCAs:            ca,
			MinVersion:         tls.VersionTLS12, // Minimum version TLS 1.2

		}
	}

	for {
		var j JobT
		select {
		case <-SignalChan:
			log.Printf("Scrape signal received, shutting down worker %d", worker)
			return
		case j = <-jobs:
			address := j.key
			moduleID := j.value
			ctx := context.Background()

			log.Print("START PASSIVE CHECK, WORKER ", worker, " (", moduleID, "), ", address, "...")

			scanner := Session.Query(`SELECT space_id, status, metric, object, data_type FROM task WHERE module_id = ?`, moduleID).WithContext(ctx).Consistency(ConsistencyRead).Iter().Scanner()
			for scanner.Next() {
				var metricValue string

				var spaceID string
				var status int
				var metric string
				var object string
				var dataType string
				err := scanner.Scan(&spaceID, &status, &metric, &object, &dataType)
				if err != nil {
					log.Print(err)
				} else {
					if status == 1 && object == "zbxagentpassive" {
						start := time.Now()

						var conn net.Conn
						var errConn error
						if AppConfig.ZBX.TLS {
							conn, errConn = tls.Dial(AppConfig.ZBX.PASSIVETYPE, address+":"+AppConfig.ZBX.PASSIVEPORT, configTLS)

							if connTLS, ok := conn.(*tls.Conn); ok {
								err := connTLS.Handshake()
								if err != nil {
									log.Printf("ZBX Client: No connection established, ERROR TLS HANDSHAKE (%v)", err)
									return
								}

								state := connTLS.ConnectionState()
								log.Printf("ZBX Client: TLS connection is established. TLS Version: %s, TLS ChipherSuite: %v, TLS Handshake: %v\n",
									tlsVersionToString(state.Version),
									state.CipherSuite,
									state.HandshakeComplete)
							}
						} else {
							conn, errConn = net.Dial(AppConfig.ZBX.PASSIVETYPE, address+":"+AppConfig.ZBX.PASSIVEPORT)
						}
						if errConn != nil {
							log.Printf("ZBX Agent NOT AVAILABLE (%v)", err)
						} else {
							var buf01 bytes.Buffer
							flags := ZBXTCPProtocol
							if ZBXCompress {
								z := zlib.NewWriter(&buf01)
								if _, err := z.Write([]byte(metric)); err != nil {
									log.Print(err)
								}
								z.Close()
								flags |= ZBXZLibCompress
							} else {
								buf01.Write([]byte(metric))
							}

							var buf02 bytes.Buffer
							buf02.Grow(buf01.Len() + ZBXHeaderSize)
							buf02.Write([]byte{'Z', 'B', 'X', 'D', flags})
							if err := binary.Write(&buf02, binary.LittleEndian, uint32(buf01.Len())); nil != err {
								log.Print(err)
							}
							if err := binary.Write(&buf02, binary.LittleEndian, uint32(len([]byte(metric)))); nil != err {
								log.Print(err)
							}
							buf02.Write(buf01.Bytes())
							sizeWrite, err := conn.Write(buf02.Bytes())
							if err != nil {
								log.Print(err)
							} else {
								log.Print("Write: ", sizeWrite)
							}

							hdr := make([]byte, 13)
							tmp := make([]byte, 1024)
							buf := make([]byte, 0)

							log.Print(conn.RemoteAddr())

							sizeRead, err := conn.Read(hdr)
							if err != nil {
								log.Print(err)
							} else {
								log.Print("ZBX Header read: ", sizeRead)

								protocolFlag := int(hdr[4])
								log.Print("ZBX Protocol FLAG: ", protocolFlag)
								protocolSize := (hdr[5:12])
								var protocolSizeInt uint32
								hdr01 := bytes.NewReader(protocolSize)
								err = binary.Read(hdr01, binary.LittleEndian, &protocolSizeInt)
								if err != nil {
									log.Print("Binary read failed: ", err)
								} else {
									buf = append(buf, hdr[:13]...)

									var dataSize uint32
									dataSize = 0
									for {
										if dataSize == protocolSizeInt+13 {
											break
										}
										sizeRead, err := conn.Read(tmp)
										if err != nil {
											log.Print("Error reading: ", err.Error())
											break
										}
										log.Print("Read: ", sizeRead)
										buf = append(buf, tmp[:sizeRead]...)
										dataSize = uint32(len(buf))
									}

									sizeRead := len(buf)
									log.Print("END Read: ", sizeRead)
									protocolData := (buf[13 : 13+protocolSizeInt])

									uncompressData, err := zbxUncompress(protocolData, protocolSizeInt, protocolFlag)
									if err != nil {
									} else {
										log.Print("ZBX PASSIVE RESPONSE: ", string(uncompressData))
										metricValue = string(uncompressData)
									}
								}
							}
							duration := time.Since(start).Seconds()
							zbxClientDuration.WithLabelValues(moduleID).Set(duration)
						}

						log.Print("PASSIVE AGENT ", moduleID, " DATA SAVING...")

						if spaceID != "" && metric != "" {
							if dataType == "float" {
								valueFloat, _ := strconv.ParseFloat(metricValue, 64)

								spaceStatus := 0
								spaceDescription := ""
								if err := Session.Query(`SELECT status, description FROM space WHERE id = ? LIMIT 1`, spaceID).WithContext(ctx).Consistency(ConsistencyRead).Scan(&spaceStatus, &spaceDescription); err != nil {
									log.Print(err)
								}
								if spaceStatus == 1 {
									log.Print("The raw_data table is used...")
									err := Session.Query(
										`BEGIN BATCH
											INSERT INTO raw_data (metric, value, create_time, event_time, space_id, object, space_description) VALUES (?, ?, toTimestamp(now()), toTimestamp(now()), ?, 'zbxagentpassive', ?)
											INSERT INTO space_metric_data (space_id, metric) VALUES (?, ?)
										APPLY BATCH`,
										metric, float32(valueFloat),
										spaceID, spaceDescription,
										spaceID, metric).WithContext(ctx).Exec()
									if err != nil {
										log.Print(err)
									} else {
										spaceIDCount, err := gocql.ParseUUID(spaceID)
										if err != nil {
											log.Print(err)
										} else {
											err = rawCount(spaceIDCount, "data", 1)
											if err != nil {
												log.Print(err)
											}
										}
										log.Print("PASSIVE DATA SAVED... OK")
										go problem(spaceID, moduleID, metric, makeTimestamp(), float32(valueFloat), "", "zbxagentpassive")
									}
								}
							} else {
								spaceStatus := 0
								spaceDescription := ""
								if err := Session.Query(`SELECT status, description FROM space WHERE id = ? LIMIT 1`, spaceID).WithContext(ctx).Consistency(ConsistencyRead).Scan(&spaceStatus, &spaceDescription); err != nil {
									log.Print(err)
								}
								if spaceStatus == 1 {
									log.Print("The raw_text table is used...")
									err := Session.Query(
										`BEGIN BATCH
											INSERT INTO raw_text (metric, value, create_time, event_time, space_id, object, space_description) VALUES (?, ?, toTimestamp(now()), toTimestamp(now()), ?, 'zbxagentpassive', ?)
											INSERT INTO space_metric_text (space_id, metric) VALUES (?, ?)
										APPLY BATCH`,
										metric, metricValue,
										spaceID, spaceDescription,
										spaceID, metric).WithContext(ctx).Exec()
									if err != nil {
										log.Print(err)
									} else {
										spaceIDCount, err := gocql.ParseUUID(spaceID)
										if err != nil {
											log.Print(err)
										} else {
											err := rawCount(spaceIDCount, "data", 1)
											if err != nil {
												log.Print(err)
											}
										}
										log.Print("PASSIVE TEXT SAVED... OK")
									}
								}
							}
							durationFull := time.Since(start).Seconds()
							zbxClientDurationFull.WithLabelValues(moduleID).Set(durationFull)
						}
					}
				}
			}
		}
	}
}

func zbxWorkersRun() {
	workerPool := AppConfig.ZBX.WORKERPOOL
	log.Printf("ZBX WORKERS START %d... %s", workerPool, " OK")

	jobs := make(chan JobT, workerPool)
	defer close(jobs)

	for worker := 1; worker <= workerPool; worker++ {
		go zbxClient(worker, jobs)
	}

	ticker := time.NewTicker(time.Duration(AppConfig.ZBX.WORKERACTIME) * time.Second)
	defer ticker.Stop()
	for ; true; <-ticker.C {
		if len(ZBXAgentMap) > 0 {
			log.Print("Summer agents for passive verification: ", len(ZBXAgentMap))
			for key, value := range ZBXAgentMap {
				var jobData JobT
				jobData.key = key
				jobData.value = value
				jobs <- jobData
			}
		}
	}
}
