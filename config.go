package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type TAppConfigDM struct {
	ID string `json:"ID"`
}

type TAppConfigTLS struct {
	CERTPATH    string `json:"CERTPATH"`
	KEYPATH     string `json:"KEYPATH"`
	KEYPASSWORD string `json:"KEYPASSWORD"`
	CAPATH      string `json:"CAPATH"`
}

type (
	// TAppConfigHTTP HTTP server launching parameters
	TAppConfigHTTP struct {
		HOST         string `json:"HOST"`
		PORT         string `json:"PORT"`
		READTIMEOUT  int    `json:"READTIMEOUT"`
		WRITETIMEOUT int    `json:"WRITETIMEOUT"`
		RL           int    `json:"RL"`
		TLS          bool   `json:"TLS"`
		SKIPVERIFY   bool   `json:"SKIPVERIFY"`
	}

	/*
		Consistensy:
		Any
		One
		Two
		Three
		Quorum
		All
		LocalQuorum
		EachQuorum
		LocalOn
	*/
	TAppConfigDB struct {
		HOSTS           []string `json:"HOSTS"`
		USERNAME        string   `json:"USERNAME"`
		PASSWORD        string   `json:"PASSWORD"`
		KEYSPACE        string   `json:"KEYSPACE"`
		CONSISTENCY     string   `json:"CONSISTENCY"`
		CONSISTENCYREAD string   `json:"CONSISTENCYREAD"`
		TIMEOUT         int      `json:"TIMEOUT"`
		TLS             bool     `json:"TLS"`
		SKIPVERIFY      bool     `json:"SKIPVERIFY"`
		BATCHSIZE       int      `json:"BATCHSIZE"`
	}

	// TAppConfigZBX ZBX server launch parameters
	TAppConfigZBX struct {
		HOST         string `json:"HOST"`
		PORT         string `json:"PORT"`
		TYPE         string `json:"TYPE"`
		PASSIVEPORT  string `json:"PASSIVEPORT"`
		PASSIVETYPE  string `json:"PASSIVETYPE"`
		TLS          bool   `json:"TLS"`
		SKIPVERIFY   bool   `json:"SKIPVERIFY"`
		WORKERPOOL   int    `json:"WORKERPOOL"`
		WORKERACTIME int    `json:"WORKERACTIME"`
	}

	// TAppConfigPRMTScraper Parameters for the Prometheus scraper
	TAppConfigPRMTScraper struct {
		ENABLED        bool          `json:"ENABLED"`
		INTERVAL       time.Duration `json:"INTERVAL"`
		TIMEOUT        time.Duration `json:"TIMEOUT"`
		RETRYATTEMPTS  int           `json:"RETRYATTEMPTS"`
		RETRYDELAY     time.Duration `json:"RETRYDELAY"`
		MAXCONNECTIONS int           `json:"MAXCONNECTIONS"`
		WORKERSPOOL    int           `json:"WORKERSPOOL"`
		TLS            bool          `json:"TLS"`
		SKIPVERIFY     bool          `json:"SKIPVERIFY"`
	}

	// TAppConfigPRMTExporter Parameters for the Prometheus exporter
	TAppConfigPRMTExporter struct {
		ENABLED     bool   `json:"ENABLED"`
		METRICSPATH string `json:"METRICSPATH"`
	}

	// TAppConfigSWG Swagger parameters
	TAppConfigSWG struct {
		TITLE       string `json:"TITLE"`
		DESCRIPTION string `json:"DESCRIPTION"`
		VERSION     string `json:"VERSION"`
		HOST        string `json:"HOST"`
		BASEPATH    string `json:"BASEPATH"`
		SCHEME      string `json:"SCHEME"`
	}

	// TAppConfigSMTP SMTP server parameters
	TAppConfigSMTP struct {
		ENABLED        bool   `json:"ENABLED"`
		HOST           string `json:"HOST"`
		PORT           int    `json:"PORT"`
		USERNAME       string `json:"USERNAME"`
		PASSWORD       string `json:"PASSWORD"`
		KEEPALIVE      bool   `json:"KEEPALIVE"`
		TIMEOUTCONNECT int    `json:"TIMEOUTCONNECT"`
		TIMEOUTSEND    int    `json:"TIMEOUTSEND"`
		TLS            bool   `json:"TLS"`
		SKIPVERIFY     bool   `json:"SKIPVERIFY"`
		FROM           string `json:"FROM"`
		QUEUE          int    `json:"QUEUE"`
		AUTH           string `json:"AUTH"`
	}

	// TAppConfig generalStructureOfServiceParameters
	TAppConfig struct {
		DM           TAppConfigDM           `json:"DM"`
		TLS          TAppConfigTLS          `json:"TLS"`
		HTTP         TAppConfigHTTP         `json:"HTTP"`
		DB           TAppConfigDB           `json:"DB"`
		ZBX          TAppConfigZBX          `json:"ZBX"`
		SWG          TAppConfigSWG          `json:"SWG"`
		SMTP         TAppConfigSMTP         `json:"SMTP"`
		PRMTEXPORTER TAppConfigPRMTExporter `json:"PRMTEXPORTER"`
		PRMTSCRAPER  TAppConfigPRMTScraper  `json:"PRMTSCRAPER"`
	}
)

var (
	// AppConfig Структура TAppConfig
	AppConfig TAppConfig
)

// LoadConfig Загрузка конфигурации, для Docker версии конфиг собирается из переменного окружения
func (appCfg *TAppConfig) LoadConfig() (err error) {

	log.Print("Download configuration from the file...")
	data, err := os.ReadFile(ConfigFile)
	if err == nil {
		err = json.Unmarshal(data, &appCfg)
		if err != nil {
			return err
		}
	} else {
		log.Print("Failed to read the configuration from the file (" + err.Error() + ")!")
	}

	/*
		Environment variables (for Docker version):
		MONITOR_DM_ID Unique identifier of the DataManager instance

		MONITOR_TLS_CERTPATH Path to certificate for main service
		MONITOR_TLS_KEYPATH Path to private key for main service
		MONITOR_TLS_KEYPASSWORD The password for the TLS private key
		MONITOR_TLS_CAPATH Path to CA certificate for main service

		MONITOR_HTTP_HOST Address for incoming API connections
		MONITOR_HTTP_PORT Port for incoming API connections
		MONITOR_HTTP_READTIMEOUT Read Timeout for HTTP server
		MONITOR_HTTP_WRITETIMEOUT Write Timeout for HTTP server
		MONITOR_HTTP_RL Requests Limit
		MONITOR_HTTP_TLS Enable TLS for HTTP server (true/false)
		MONITOR_HTTP_TLS_SKIPVERIFY Enable insecure skip verify for HTTP server (true/false)

		MONITOR_DB_HOST Array of addresses for DB connection, comma-separated (127.0.0.1:19042,127.0.0.1:29042)
		MONITOR_DB_USERNAME Username for DB connection
		MONITOR_DB_PASSWORD Password for DB connection
		MONITOR_DB_TLS Enable TLS for DB cluster connection (true/false)
		MONITOR_DB_TLS_SKIPVERIFY Disable HostVerification by certificate for DB connection (true/false)
		MONITOR_DB_KEYSPACE Keyspace for DB connection
		MONITOR_DB_CONSISTENCY Consistency parameter for data in DB cluster
		MONITOR_DB_CONSISTENCYREAD Consistency parameter for reading data from DB
		MONITOR_DB_TIMEOUT Query execution timeout before forced termination
		MONITOR_BATCHSIZE Batch sizes are formed by the application

		MONITOR_ZBX_HOST Address for incoming ZBX server connections
		MONITOR_ZBX_PORT Port for incoming ZBX server connections
		MONITOR_ZBX_TYPE ZBX server protocol
		MONITOR_ZBX_PASSIVEPORT Port of a passive agent
		MONITOR_ZBX_PASSIVETYPE ZBC client protocol
		MONITOR_ZBX_TLS Enable TLS for ZBX agent connection (true/false)
		MONITOR_ZBX_SKIPVERIFY Enable insecure skip verify for ZBX agent connection (true/false)
		MONITOR_ZBX_WORKER_POOL Number of workers started for active server checks
		MONITOR_ZBX_WORKER_ACTIME Interval for running active server checks in seconds

		MONITOR_PRMTEXPORTER_ENABLED Enable Prometheus exporter (true/false)
		MONITOR_PRMTEXPORTER_METRICSPATH Path to metrics for Prometheus exporter

		MONITOR_PRMTSCRAPER_ENABLED Enable Prometheus scraper (true/false)
		MONITOR_PRMTSCRAPER_INTERVAL Interval for Prometheus scraper in seconds
		MONITOR_PRMTSCRAPER_TIMEOUT Timeout for Prometheus scraper in seconds
		MONITOR_PRMTSCRAPER_RETRYATTEMPTS Number of retry attempts for Prometheus scraper
		MONITOR_PRMTSCRAPER_RETRYDELAY Delay between retry attempts for Prometheus scraper in seconds
		MONITOR_PRMTSCRAPER_MAXCONNECTIONS Maximum number of simultaneous connections for Prometheus scraper
		MONITOR_PRMTSCRAPER_WORKERSPOOL Number of workers for Prometheus scraper
		MONITOR_PRMTSCRAPER_TLS Enable TLS for Prometheus scraper (true/false)
		MONITOR_PRMTSCRAPER_TLS_SKIPVERIFY Disable insecure skip verify for Prometheus scraper (true/false)

		MONITOR_SWG_TITLE Title on Swagger page
		MONITOR_SWG_DESCRIPTION Description on Swagger page
		MONITOR_SWG_VERSION API version on Swagger page
		MONITOR_SWG_HOST Full API address for Swagger
		MONITOR_SWG_BASEPATH Base API path for Swagger
		MONITOR_SWG_SCHEME Base protocol for Swagger

		MONITOR_SMTP_ENABLED Enable SMTP server (true/false)
		MONITOR_SMTP_HOST SMTP server address
		MONITOR_SMTP_PORT SMTP server port
		MONITOR_SMTP_USERNAME Username for SMTP server connection
		MONITOR_SMTP_PASSWORD Password for SMTP server connection
		MONITOR_SMTP_KEEPALIVE Use KeepAlive connection with SMTP server (true/false)
		MONITOR_SMTP_TIMEOUTCONNECT SMTP server connection timeout in seconds
		MONITOR_SMTP_TIMEOUTSEND SMTP server send timeout in seconds
		MONITOR_SMTP_TLS Use TLS connection with SMTP server (true/false)
		MONITOR_SMTP_TLS_SKIPVERIFY Disable insecure skip verify for Prometheus scraper (true/false)
		MONITOR_SMTP_AUTH Тап авторизации
		MONITOR_SMTP_FROM From address for SMTP server
		MONITOR_SMTP_QUEUE SMTP queue size
	*/

	log.Print("Loading configuration from alternating environment...")

	// DM
	var dmIDENV string
	dmIDENV, _ = os.LookupEnv("MONITOR_DM_ID")
	if dmIDENV != "" {
		appCfg.DM.ID = dmIDENV
	}

	// TLS for main service
	var tlsCertPathENV string
	tlsCertPathENV, _ = os.LookupEnv("MONITOR_TLS_CERTPATH")
	if tlsCertPathENV != "" {
		appCfg.TLS.CERTPATH = tlsCertPathENV
	}
	var tlsKeyPathENV string
	tlsKeyPathENV, _ = os.LookupEnv("MONITOR_TLS_KEYPATH")
	if tlsKeyPathENV != "" {
		appCfg.TLS.KEYPATH = tlsKeyPathENV
	}
	var tlsKeyPasswordENV string
	tlsKeyPasswordENV, _ = os.LookupEnv("MONITOR_TLS_KEYPASSWORD")
	if tlsKeyPasswordENV != "" {
		appCfg.TLS.KEYPASSWORD = tlsKeyPasswordENV
	}
	var tlsCAPathENV string
	tlsCAPathENV, _ = os.LookupEnv("MONITOR_TLS_CAPATH")
	if tlsCAPathENV != "" {
		appCfg.TLS.CAPATH = tlsCAPathENV
	}

	// The main API + Prometheus Exporter HTTP server
	var httpHostENV string
	httpHostENV, _ = os.LookupEnv("MONITOR_HTTP_HOST")
	if httpHostENV != "" {
		appCfg.HTTP.HOST = httpHostENV
	}
	var httpPortENV string
	httpPortENV, _ = os.LookupEnv("MONITOR_HTTP_PORT")
	if httpPortENV != "" {
		appCfg.HTTP.PORT = httpPortENV
	}
	var httpReadTimeoutENV string
	httpReadTimeoutENV, _ = os.LookupEnv("MONITOR_HTTP_READTIMEOUT")
	if httpReadTimeoutENV != "" {
		var httpReadTimeoutINT int
		httpReadTimeoutINT, err = strconv.Atoi(httpReadTimeoutENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_HTTP_READTIMEOUT (" + err.Error() + ")!")
		} else {
			appCfg.HTTP.READTIMEOUT = httpReadTimeoutINT
		}
	}
	var httpWriteTimeoutENV string
	httpWriteTimeoutENV, _ = os.LookupEnv("MONITOR_HTTP_READTIMEOUT")
	if httpWriteTimeoutENV != "" {
		var httpWriteTimeoutINT int
		httpWriteTimeoutINT, err = strconv.Atoi(httpWriteTimeoutENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_HTTP_WRITETIMEOUT (" + err.Error() + ")!")
		} else {
			appCfg.HTTP.WRITETIMEOUT = httpWriteTimeoutINT
		}
	}
	var httpRLENV string
	httpRLENV, _ = os.LookupEnv("MONITOR_HTTP_RL")
	if httpRLENV != "" {
		var httpRLINT int
		httpRLINT, err = strconv.Atoi(httpRLENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_HTTP_RL (" + err.Error() + ")!")
		} else {
			appCfg.HTTP.RL = httpRLINT
		}
	}
	var httpTLSENV string
	httpTLSENV, _ = os.LookupEnv("MONITOR_HTTP_TLS")
	if httpTLSENV != "" {
		boolValue, err := strconv.ParseBool(httpTLSENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_HTTP_TLS (" + err.Error() + ")!")
		} else {
			appCfg.HTTP.TLS = boolValue
		}
	}
	var httpSkipVerifyENV string
	httpSkipVerifyENV, _ = os.LookupEnv("MONITOR_HTTP_TLS_SKIPVERIFY")
	if httpSkipVerifyENV != "" {
		boolValue, err := strconv.ParseBool(httpSkipVerifyENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_HTTP_TLS_SKIPVERIFY (" + err.Error() + ")!")
		} else {
			appCfg.HTTP.SKIPVERIFY = boolValue
		}
	}

	// Connection to the database
	var dbHostsENV string
	dbHostsENV, _ = os.LookupEnv("MONITOR_DB_HOST")
	if dbHostsENV != "" {
		dbHostsENV = strings.TrimSpace(dbHostsENV)       // Удаление возможных пробелов
		appCfg.DB.HOSTS = strings.Split(dbHostsENV, ",") // Разбиваем строку по символу ","
	}
	var dbUserNameENV string
	dbUserNameENV, _ = os.LookupEnv("MONITOR_DB_USERNAME")
	if dbUserNameENV != "" {
		appCfg.DB.USERNAME = dbUserNameENV
	}
	var dbPasswordENV string
	dbPasswordENV, _ = os.LookupEnv("MONITOR_DB_PASSWORD")
	if dbPasswordENV != "" {
		appCfg.DB.PASSWORD = dbPasswordENV
	}
	var dbKeySpaceENV string
	dbKeySpaceENV, _ = os.LookupEnv("MONITOR_DB_KEYSPACE")
	if dbKeySpaceENV != "" {
		appCfg.DB.KEYSPACE = dbKeySpaceENV
	}
	var dbConsistensyENV string
	dbConsistensyENV, _ = os.LookupEnv("MONITOR_DB_CONSISTENCY")
	if dbConsistensyENV != "" {
		appCfg.DB.CONSISTENCY = dbConsistensyENV
	}
	var dbConsistensyReadENV string
	dbConsistensyReadENV, _ = os.LookupEnv("MONITOR_DB_CONSISTENCYREAD")
	if dbConsistensyReadENV != "" {
		appCfg.DB.CONSISTENCYREAD = dbConsistensyReadENV
	}
	var dbTimeoutENV string
	dbTimeoutENV, _ = os.LookupEnv("MONITOR_DB_TIMEOUT")
	if dbTimeoutENV != "" {
		var dbTimeoutINT int
		dbTimeoutINT, err = strconv.Atoi(dbTimeoutENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_DB_TIMEOUT (" + err.Error() + ")!")
		} else {
			appCfg.DB.TIMEOUT = dbTimeoutINT
		}
	}
	var dbTLSENV string
	dbTLSENV, _ = os.LookupEnv("MONITOR_DB_TLS")
	if dbTLSENV != "" {
		boolValue, err := strconv.ParseBool(dbTLSENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_DB_TLS (" + err.Error() + ")!")
		} else {
			appCfg.DB.TLS = boolValue
		}
	}
	var dbSkipVerifyENV string
	dbSkipVerifyENV, _ = os.LookupEnv("MONITOR_DB_TLS_SKIPVERIFY")
	if dbSkipVerifyENV != "" {
		boolValue, err := strconv.ParseBool(dbSkipVerifyENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_DB_HOSTVERIFICATION (" + err.Error() + ")!")
		} else {
			appCfg.DB.SKIPVERIFY = boolValue
		}
	}
	var dbBatchSizeENV string
	dbBatchSizeENV, _ = os.LookupEnv("MONITOR_DB_BATCHSIZE")
	if dbBatchSizeENV != "" {
		var dbBatchSizeINT int
		dbBatchSizeINT, err = strconv.Atoi(dbBatchSizeENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_DB_BATCHSIZE (" + err.Error() + ")!")
		} else {
			appCfg.DB.BATCHSIZE = dbBatchSizeINT
		}
	}

	// ZBX server
	var zbxHostENV string
	zbxHostENV, _ = os.LookupEnv("MONITOR_ZBX_HOST")
	if zbxHostENV != "" {
		appCfg.ZBX.HOST = zbxHostENV
	}
	var zbxPortENV string
	zbxPortENV, _ = os.LookupEnv("MONITOR_ZBX_PORT")
	if zbxPortENV != "" {
		appCfg.ZBX.PORT = zbxPortENV
	}
	var zbxTypeENV string
	zbxTypeENV, _ = os.LookupEnv("MONITOR_ZBX_TYPE")
	if zbxTypeENV != "" {
		appCfg.ZBX.TYPE = zbxTypeENV
	}

	// ZBX Client
	var zbxPassivePortENV string
	zbxPassivePortENV, _ = os.LookupEnv("MONITOR_ZBX_PASSIVEPORT")
	if zbxPassivePortENV != "" {
		appCfg.ZBX.PASSIVEPORT = zbxPassivePortENV
	}
	var zbxPassiveTypeENV string
	zbxPassiveTypeENV, _ = os.LookupEnv("MONITOR_ZBX_PASSIVETYPE")
	if zbxPassiveTypeENV != "" {
		appCfg.ZBX.PASSIVETYPE = zbxPassiveTypeENV
	}

	// ZBX Worker
	var zbxWorkerPoolENV string
	zbxWorkerPoolENV, _ = os.LookupEnv("MONITOR_ZBX_WORKER_POOL")
	if zbxWorkerPoolENV != "" {
		var zbxWorkerPoolINT int
		zbxWorkerPoolINT, err = strconv.Atoi(zbxWorkerPoolENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_ZBX_WORKER_POOL (" + err.Error() + ")!")
		} else {
			appCfg.ZBX.WORKERPOOL = zbxWorkerPoolINT
		}
	}
	var zbxWorkerACTimeENV string
	zbxWorkerACTimeENV, _ = os.LookupEnv("MONITOR_ZBX_WORKER_ACTIME")
	if zbxWorkerACTimeENV != "" {
		var zbxWorkerACTimeINT int
		zbxWorkerACTimeINT, err = strconv.Atoi(zbxWorkerACTimeENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_ZBX_WORKER_ACTIME (" + err.Error() + ")!")
		} else {
			appCfg.ZBX.WORKERACTIME = zbxWorkerACTimeINT
		}
	}

	// ZBX TLS
	var zbxTLSENV string
	zbxTLSENV, _ = os.LookupEnv("MONITOR_ZBX_TLS")
	if zbxTLSENV != "" {
		boolValue, err := strconv.ParseBool(zbxTLSENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_ZBX_TLS (" + err.Error() + ")!")
		} else {
			appCfg.ZBX.TLS = boolValue
		}
	}
	var zbxSkipVerifyENV string
	zbxSkipVerifyENV, _ = os.LookupEnv("MONITOR_ZBX_TLS_SKIPVERIFY")
	if zbxSkipVerifyENV != "" {
		boolValue, err := strconv.ParseBool(zbxSkipVerifyENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_ZBX_TLS_SKIPVERIFY (" + err.Error() + ")!")
		} else {
			appCfg.ZBX.SKIPVERIFY = boolValue
		}
	}

	// Prometheus Exporter
	var prmtExporterEnabledENV string
	prmtExporterEnabledENV, _ = os.LookupEnv("MONITOR_PRMTEXPORTER_ENABLED")
	if prmtExporterEnabledENV != "" {
		appCfg.PRMTEXPORTER.ENABLED, err = strconv.ParseBool(prmtExporterEnabledENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_PRMTEXPORTER_ENABLED (" + err.Error() + ")!")
			appCfg.PRMTEXPORTER.ENABLED = false
		}
	}
	var prmtExporterMetricsPathENV string
	prmtExporterMetricsPathENV, _ = os.LookupEnv("MONITOR_PRMTEXPORTER_METRICSPATH")
	if prmtExporterMetricsPathENV != "" {
		appCfg.PRMTEXPORTER.METRICSPATH = prmtExporterMetricsPathENV
	}

	// Prometheus Scraper
	var prmtScraperEnabledENV string
	prmtScraperEnabledENV, _ = os.LookupEnv("MONITOR_PRMTSCRAPER_ENABLED")
	if prmtScraperEnabledENV != "" {
		appCfg.PRMTSCRAPER.ENABLED, err = strconv.ParseBool(prmtScraperEnabledENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_PRMTSCRAPER_ENABLED (" + err.Error() + ")!")
			appCfg.PRMTSCRAPER.ENABLED = false
		}
	}
	var prmtScraperIntervalENV string
	prmtScraperIntervalENV, _ = os.LookupEnv("MONITOR_PRMTSCRAPER_INTERVAL")
	if prmtScraperIntervalENV != "" {
		var prmtScraperIntervalINT int
		prmtScraperIntervalINT, err = strconv.Atoi(prmtScraperIntervalENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_PRMTSCRAPER_INTERVAL (" + err.Error() + ")!")
		} else {
			appCfg.PRMTSCRAPER.INTERVAL = time.Duration(prmtScraperIntervalINT) * time.Second
		}
	}
	var prmtScraperTimeoutENV string
	prmtScraperTimeoutENV, _ = os.LookupEnv("MONITOR_PRMTSCRAPER_TIMEOUT")
	if prmtScraperTimeoutENV != "" {
		var prmtScraperTimeoutINT int
		prmtScraperTimeoutINT, err = strconv.Atoi(prmtScraperTimeoutENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_PRMTSCRAPER_TIMEOUT (" + err.Error() + ")!")
		} else {
			appCfg.PRMTSCRAPER.TIMEOUT = time.Duration(prmtScraperTimeoutINT) * time.Second
		}
	}
	var prmtScraperRetryAttemptsENV string
	prmtScraperRetryAttemptsENV, _ = os.LookupEnv("MONITOR_PRMTSCRAPER_RETRYATTEMPTS")
	if prmtScraperRetryAttemptsENV != "" {
		var prmtScraperRetryAttemptsINT int
		prmtScraperRetryAttemptsINT, err = strconv.Atoi(prmtScraperRetryAttemptsENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_PRMTSCRAPER_RETRYATTEMPTS (" + err.Error() + ")!")
		} else {
			appCfg.PRMTSCRAPER.RETRYATTEMPTS = prmtScraperRetryAttemptsINT
		}
	}
	var prmtScraperRetryDelayENV string
	prmtScraperRetryDelayENV, _ = os.LookupEnv("MONITOR_PRMTSCRAPER_RETRYDELAY")
	if prmtScraperRetryDelayENV != "" {
		var prmtScraperRetryDelayINT int
		prmtScraperRetryDelayINT, err = strconv.Atoi(prmtScraperRetryDelayENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_PRMTSCRAPER_RETRYDELAY (" + err.Error() + ")!")
		} else {
			appCfg.PRMTSCRAPER.RETRYDELAY = time.Duration(prmtScraperRetryDelayINT) * time.Second
		}
	}
	var prmtScraperMaxConnectionsENV string
	prmtScraperMaxConnectionsENV, _ = os.LookupEnv("MONITOR_PRMTSCRAPER_MAXCONNECTIONS")
	if prmtScraperMaxConnectionsENV != "" {
		var prmtScraperMaxConnectionsINT int
		prmtScraperMaxConnectionsINT, err = strconv.Atoi(prmtScraperMaxConnectionsENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_PRMTSCRAPER_MAXCONNECTIONS (" + err.Error() + ")!")
		} else {
			appCfg.PRMTSCRAPER.MAXCONNECTIONS = prmtScraperMaxConnectionsINT
		}
	}
	var prmtScraperWorkersPoolENV string
	prmtScraperWorkersPoolENV, _ = os.LookupEnv("MONITOR_PRMTSCRAPER_WORKERSPOOL")
	if prmtScraperWorkersPoolENV != "" {
		var prmtScraperWorkersPoolINT int
		prmtScraperWorkersPoolINT, err = strconv.Atoi(prmtScraperWorkersPoolENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_PRMTSCRAPER_WORKERSPOOL (" + err.Error() + ")!")
			appCfg.PRMTSCRAPER.WORKERSPOOL = 1
		} else {
			appCfg.PRMTSCRAPER.WORKERSPOOL = prmtScraperWorkersPoolINT
		}
	}
	var prmtScraperTLSENV string
	prmtScraperTLSENV, _ = os.LookupEnv("MONITOR_PRMTSCRAPER_TLS")
	if prmtScraperTLSENV != "" {
		boolValue, err := strconv.ParseBool(prmtScraperTLSENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_PRMTSCRAPER_TLS (" + err.Error() + ")!")
		} else {
			appCfg.PRMTSCRAPER.TLS = boolValue
		}
	}
	var prmtScraperSkipVerifyENV string
	prmtScraperSkipVerifyENV, _ = os.LookupEnv("MONITOR_PRMTSCRAPER_TLS_SKIPVERIFY")
	if prmtScraperSkipVerifyENV != "" {
		boolValue, err := strconv.ParseBool(prmtScraperSkipVerifyENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_PRMTSCRAPER_TLS_SKIPVERIFY (" + err.Error() + ")!")
		} else {
			appCfg.PRMTSCRAPER.SKIPVERIFY = boolValue
		}
	}

	// Swagger
	var swgTitleENV string
	swgTitleENV, _ = os.LookupEnv("MONITOR_SWG_TITLE")
	if swgTitleENV != "" {
		appCfg.SWG.TITLE = swgTitleENV
	}
	var swgDescriptionENV string
	swgDescriptionENV, _ = os.LookupEnv("MONITOR_SWG_DESCRIPTION")
	if swgDescriptionENV != "" {
		appCfg.SWG.DESCRIPTION = swgDescriptionENV
	}
	var swgVersionENV string
	swgVersionENV, _ = os.LookupEnv("MONITOR_SWG_VERSION")
	if swgVersionENV != "" {
		appCfg.SWG.VERSION = swgVersionENV
	}
	var swgHostENV string
	swgHostENV, _ = os.LookupEnv("MONITOR_SWG_HOST")
	if swgHostENV != "" {
		appCfg.SWG.HOST = swgHostENV
	}
	var swgBasePathENV string
	swgBasePathENV, _ = os.LookupEnv("MONITOR_SWG_BASEPATH")
	if swgBasePathENV != "" {
		appCfg.SWG.BASEPATH = swgBasePathENV
	}
	var swgSchemeENV string
	swgSchemeENV, _ = os.LookupEnv("MONITOR_SWG_SCHEME")
	if swgSchemeENV != "" {
		appCfg.SWG.SCHEME = swgSchemeENV
	}

	// SMTP server
	var smtpEnabledENV string
	smtpEnabledENV, _ = os.LookupEnv("MONITOR_SMTP_ENABLED")
	if smtpEnabledENV != "" {
		boolValue, err := strconv.ParseBool(smtpEnabledENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_SMTP_ENABLED (" + err.Error() + ")!")
		} else {
			appCfg.SMTP.ENABLED = boolValue
		}
	}
	var smtpHostENV string
	smtpHostENV, _ = os.LookupEnv("MONITOR_SMTP_HOST")
	if smtpHostENV != "" {
		appCfg.SMTP.HOST = smtpHostENV
	}
	var smtpPortENV string
	smtpPortENV, _ = os.LookupEnv("MONITOR_SMTP_PORT")
	if smtpPortENV != "" {
		var smtpPortINT int
		smtpPortINT, err = strconv.Atoi(smtpPortENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_SMTP_PORT (" + err.Error() + ")!")
		} else {
			appCfg.SMTP.PORT = smtpPortINT
		}
	}
	var smtpUserNameENV string
	smtpUserNameENV, _ = os.LookupEnv("MONITOR_SMTP_USERNAME")
	if smtpUserNameENV != "" {
		appCfg.SMTP.USERNAME = smtpUserNameENV
	}
	var smtpPasswordENV string
	smtpPasswordENV, _ = os.LookupEnv("MONITOR_SMTP_PASSWORD")
	if smtpPasswordENV != "" {
		appCfg.SMTP.PASSWORD = smtpPasswordENV
	}
	var smtpKeepAliveENV string
	smtpKeepAliveENV, _ = os.LookupEnv("MONITOR_SMTP_KEEPALIVE")
	if smtpKeepAliveENV != "" {
		boolValue, err := strconv.ParseBool(smtpKeepAliveENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_SMTP_KEEPALIVE (" + err.Error() + ")!")
		} else {
			appCfg.SMTP.KEEPALIVE = boolValue
		}
	}
	var smtpTimeoutConnectENV string
	smtpTimeoutConnectENV, _ = os.LookupEnv("MONITOR_SMTP_TIMEOUTCONNECT")
	if smtpTimeoutConnectENV != "" {
		var smtpTimeoutConnectINT int
		smtpTimeoutConnectINT, err = strconv.Atoi(smtpTimeoutConnectENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_SMTP_TIMEOUTCONNECT (" + err.Error() + ")!")
		} else {
			appCfg.SMTP.TIMEOUTCONNECT = smtpTimeoutConnectINT
		}
	}
	var smtpTimeoutSendENV string
	smtpTimeoutSendENV, _ = os.LookupEnv("MONITOR_SMTP_TIMEOUTSEND")
	if smtpTimeoutSendENV != "" {
		var smtpTimeoutSendINT int
		smtpTimeoutSendINT, err = strconv.Atoi(smtpTimeoutSendENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_SMTP_TIMEOUTSEND (" + err.Error() + ")!")
		} else {
			appCfg.SMTP.TIMEOUTSEND = smtpTimeoutSendINT
		}
	}
	var smtpTLSEnv string
	smtpTLSEnv, _ = os.LookupEnv("MONITOR_SMTP_TLS")
	if smtpTLSEnv != "" {
		boolValue, err := strconv.ParseBool(smtpTLSEnv)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_SMTP_TLS (" + err.Error() + ")!")
		} else {
			appCfg.SMTP.TLS = boolValue
		}
	}
	var smtpSkipVerifyENV string
	smtpSkipVerifyENV, _ = os.LookupEnv("MONITOR_SMTP_TLS_SKIPVERIFY")
	if smtpSkipVerifyENV != "" {
		boolValue, err := strconv.ParseBool(smtpSkipVerifyENV)
		if err != nil {
			log.Print("It was not possible to convert CMONITOR_SMTP_TLS_SKIPVERIFY (" + err.Error() + ")!")
		} else {
			appCfg.SMTP.SKIPVERIFY = boolValue
		}
	}
	var smtpFromENV string
	smtpFromENV, _ = os.LookupEnv("MONITOR_SMTP_FROM")
	if smtpFromENV != "" {
		appCfg.SMTP.FROM = smtpFromENV
	}
	var smtpQueueENV string
	smtpQueueENV, _ = os.LookupEnv("MONITOR_SMTP_QUEUE")
	if smtpQueueENV != "" {
		var smtpQueueINT int
		smtpQueueINT, err = strconv.Atoi(smtpQueueENV)
		if err != nil {
			log.Print("It was not possible to convert MONITOR_SMTP_QUEUE (" + err.Error() + ")!")
		} else {
			appCfg.SMTP.QUEUE = smtpQueueINT
		}
	}
	var smtpAuthENV string
	smtpAuthENV, _ = os.LookupEnv("MONITOR_SMTP_AUTH")
	if smtpAuthENV != "" {
		appCfg.SMTP.AUTH = smtpAuthENV
	}

	log.Print("Configuration parameters:")
	log.Print("DM.ID: ", appCfg.DM.ID)

	log.Print("TLS.CERTPATH: ", appCfg.TLS.CERTPATH)
	log.Print("TLS.KEYPATH: ", appCfg.TLS.KEYPATH)
	log.Print("TLS.CAPATH: ", appCfg.TLS.CAPATH)

	log.Print("HTTP.HOST: ", appCfg.HTTP.HOST)
	log.Print("HTTP.PORT: ", appCfg.HTTP.PORT)
	log.Print("HTTP.READTIMEOUT: ", appCfg.HTTP.READTIMEOUT)
	log.Print("HTTP.WRITETIMEOUT: ", appCfg.HTTP.WRITETIMEOUT)
	log.Print("HTTP.RL: ", appCfg.HTTP.RL)
	log.Print("HTTP.TLS: ", appCfg.HTTP.TLS)
	log.Print("HTTP.SKIPVERIFY: ", appCfg.HTTP.SKIPVERIFY)

	log.Print("DB.HOSTS: ", appCfg.DB.HOSTS)
	log.Print("DB.USERNAME: ", appCfg.DB.USERNAME)
	log.Print("DB.TLS: ", appCfg.DB.TLS)
	log.Print("DB.SKIPVERIFY: ", appCfg.DB.SKIPVERIFY)
	log.Print("DB.KEYSPACE: ", appCfg.DB.KEYSPACE)
	log.Print("DB.CONSISTENCY: ", appCfg.DB.CONSISTENCY)
	log.Print("DB.CONSISTENCYREAD: ", appCfg.DB.CONSISTENCYREAD)
	log.Print("DB.TIMEOUT: ", appCfg.DB.TIMEOUT)
	log.Print("DB.BATCHSIZE: ", appCfg.DB.BATCHSIZE)

	log.Print("ZBX.HOST: ", appCfg.ZBX.HOST)
	log.Print("ZBX.PORT: ", appCfg.ZBX.PORT)
	log.Print("ZBX.TYPE: ", appCfg.ZBX.TYPE)
	log.Print("ZBX.PASSIVEPORT: ", appCfg.ZBX.PASSIVEPORT)
	log.Print("ZBX.PASSIVETYPE: ", appCfg.ZBX.PASSIVETYPE)
	log.Print("ZBX.TLS: ", appCfg.ZBX.TLS)
	log.Print("ZBX.SKIPVERIFY: ", appCfg.ZBX.SKIPVERIFY)
	log.Print("ZBX.WORKERPOOL: ", appCfg.ZBX.WORKERPOOL)
	log.Print("ZBX.WORKERACTIME: ", appCfg.ZBX.WORKERACTIME)

	log.Print("PRMTEXPORTER.ENABLED: ", appCfg.PRMTEXPORTER.ENABLED)
	log.Print("PRMTEXPORTER.METRICSPATH: ", appCfg.PRMTEXPORTER.METRICSPATH)

	log.Print("PRMTSCRAPER.ENABLED: ", appCfg.PRMTSCRAPER.ENABLED)
	log.Print("PRMTSCRAPER.INTERVAL: ", appCfg.PRMTSCRAPER.INTERVAL)
	log.Print("PRMTSCRAPER.TIMEOUT: ", appCfg.PRMTSCRAPER.TIMEOUT)
	log.Print("PRMTSCRAPER.RETRYATTEMPTS: ", appCfg.PRMTSCRAPER.RETRYATTEMPTS)
	log.Print("PRMTSCRAPER.RETRYDELAY: ", appCfg.PRMTSCRAPER.RETRYDELAY)
	log.Print("PRMTSCRAPER.MAXCONNECTIONS: ", appCfg.PRMTSCRAPER.MAXCONNECTIONS)
	log.Print("PRMTSCRAPER.WORKERSPOOL: ", appCfg.PRMTSCRAPER.WORKERSPOOL)

	log.Print("SWG.TITLE: ", appCfg.SWG.TITLE)
	log.Print("SWG.DESCRIPTION: ", appCfg.SWG.DESCRIPTION)
	log.Print("SWG.VERSION: ", appCfg.SWG.VERSION)
	log.Print("SWG.HOST: ", appCfg.SWG.HOST)
	log.Print("SWG.BASEPATH: ", appCfg.SWG.BASEPATH)
	log.Print("SWG.SCHEME: ", appCfg.SWG.SCHEME)

	log.Print("SMTP.ENABLED: ", appCfg.SMTP.ENABLED)
	log.Print("SMTP.HOST: ", appCfg.SMTP.HOST)
	log.Print("SMTP.PORT: ", appCfg.SMTP.PORT)
	log.Print("SMTP.USERNAME: ", appCfg.SMTP.USERNAME)
	log.Print("SMTP.KEEPALIVE: ", appCfg.SMTP.KEEPALIVE)
	log.Print("SMTP.TIMEOUTCONNECT: ", appCfg.SMTP.TIMEOUTCONNECT)
	log.Print("SMTP.TIMEOUTSEND: ", appCfg.SMTP.TIMEOUTSEND)
	log.Print("SMTP.TLS: ", appCfg.SMTP.TLS)
	log.Print("SMTP.SKIPVERIFY: ", appCfg.SMTP.SKIPVERIFY)
	log.Print("SMTP.AUTH: ", appCfg.SMTP.AUTH)
	log.Print("SMTP.FROM: ", appCfg.SMTP.FROM)
	log.Print("SMTP.QUEUE: ", appCfg.SMTP.QUEUE)

	return nil
}
