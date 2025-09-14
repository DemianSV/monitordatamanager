package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
)

type (
	// TAppConfigHTTP HTTP server launching parameters
	TAppConfigHTTP struct {
		HOST         string `json:"HOST"`
		PORT         string `json:"PORT"`
		READTIMEOUT  int    `json:"READTIMEOUT"`
		WRITETIMEOUT int    `json:"WRITETIMEOUT"`
		RL           int    `json:"RL"`
		TLS          bool   `json:"TLS"`
		CERTPATH     string `json:"CERTPATH"`
		KEYPATH      string `json:"KEYPATH"`
		CAPATH       string `json:"CAPATH"`
	}
	TAppConfigWorker struct {
		POOL   int `json:"POOL"`
		ACTIME int `json:"ACTIME"`
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
		HOSTS            []string `json:"HOSTS"`
		USERNAME         string   `json:"USERNAME"`
		PASSWORD         string   `json:"PASSWORD"`
		CERTPATH         string   `json:"CERTPATH"`
		KEYPATH          string   `json:"KEYPATH"`
		CAPATH           string   `json:"CAPATH"`
		KEYSPACE         string   `json:"KEYSPACE"`
		CONSISTENCY      string   `json:"CONSISTENCY"`
		CONSISTENCYREAD  string   `json:"CONSISTENCYREAD"`
		TIMEOUT          int      `json:"TIMEOUT"`
		TLS              bool     `json:"TLS"`
		HOSTVERIFICATION bool     `json:"HOSTVERIFICATION"`
	}

	// TAppConfigZBX ZBX server launch parameters
	TAppConfigZBX struct {
		HOST        string `json:"HOST"`
		PORT        string `json:"PORT"`
		TYPE        string `json:"TYPE"`
		PASSIVEPORT string `json:"PASSIVEPORT"`
		PASSIVETYPE string `json:"PASSIVETYPE"`
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
		ENABLED          bool   `json:"ENABLED"`
		HOST             string `json:"HOST"`
		PORT             int    `json:"PORT"`
		USERNAME         string `json:"USERNAME"`
		PASSWORD         string `json:"PASSWORD"`
		KEEPALIVE        bool   `json:"KEEPALIVE"`
		TIMEOUTCONNECT   int    `json:"TIMEOUTCONNECT"`
		TIMEOUTSEND      int    `json:"TIMEOUTSEND"`
		TLS              bool   `json:"TLS"`
		HOSTVERIFICATION bool   `json:"HOSTVERIFICATION"`
		FROM             string `json:"FROM"`
		QUEUE            int    `json:"QUEUE"`
	}

	// TAppConfig generalStructureOfServiceParameters
	TAppConfig struct {
		HTTP   TAppConfigHTTP   `json:"HTTP"`
		WORKER TAppConfigWorker `json:"WORKER"`
		DB     TAppConfigDB     `json:"DB"`
		ZBX    TAppConfigZBX    `json:"ZBX"`
		SWG    TAppConfigSWG    `json:"SWG"`
		SMTP   TAppConfigSMTP   `json:"SMTP"`
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

		CHRDS_HTTP_HOST Address for incoming API connections
		CHRDS_HTTP_PORT Port for incoming API connections
		CHRDS_HTTP_READTIMEOUT Read Timeout for HTTP server
		CHRDS_HTTP_WRITETIMEOUT Write Timeout for HTTP server
		CHRDS_HTTP_RL Requests Limit
		CHRDS_HTTP_TLS Enable TLS for HTTP server (true/false)
		CHRDS_HTTP_CERTPATH Path to certificate for HTTP server
		CHRDS_HTTP_KEYPATH Path to private key for HTTP server
		CHRDS_HTTP_CAPATH Path to CA certificate for HTTP server

		CHRDS_WORKER_POOL Number of workers started for active server checks
		CHRDS_WORKER_ACTIME Interval for running active server checks in seconds

		CHRDS_DB_HOST Array of addresses for DB connection, comma-separated (127.0.0.1:19042,127.0.0.1:29042)
		CHRDS_DB_USERNAME Username for DB connection
		CHRDS_DB_PASSWORD Password for DB connection
		CHRDS_DB_TLS Enable TLS for DB cluster connection (true/false)
		CHRDS_DB_CERTPATH Path to certificate for DB connection
		CHRDS_DB_KEYPATH Path to private key for DB connection
		CHRDS_DB_CAPATH Path to CA certificate for DB connection
		CHRDS_DB_HOSTVERIFICATION Enable HostVerification by certificate for DB connection (true/false)
		CHRDS_DB_KEYSPACE Keyspace for DB connection
		CHRDS_DB_CONSISTENCY Consistency parameter for data in DB cluster
		CHRDS_DB_CONSISTENCYREAD Consistency parameter for reading data from DB
		CHRDS_DB_TIMEOUT Query execution timeout before forced termination

		CHRDS_ZBX_HOST Address for incoming ZBX server connections
		CHRDS_ZBX_PORT Port for incoming ZBX server connections
		CHRDS_ZBX_TYPE ZBX server protocol
		CHRDS_ZBX_PASSIVEPORT Port of a passive agent
		CHRDS_ZBX_PASSIVETYPE ZBC client protocol

		CHRDS_SWG_TITLE Title on Swagger page
		CHRDS_SWG_DESCRIPTION Description on Swagger page
		CHRDS_SWG_VERSION API version on Swagger page
		CHRDS_SWG_HOST Full API address for Swagger
		CHRDS_SWG_BASEPATH Base API path for Swagger
		CHRDS_SWG_SCHEME Base protocol for Swagger

		CHRDS_SMTP_ENABLED Enable SMTP server (true/false)
		CHRDS_SMTP_HOST SMTP server address
		CHRDS_SMTP_PORT SMTP server port
		CHRDS_SMTP_USERNAME Username for SMTP server connection
		CHRDS_SMTP_PASSWORD Password for SMTP server connection
		CHRDS_SMTP_KEEPALIVE Use KeepAlive connection with SMTP server (true/false)
		CHRDS_SMTP_TIMEOUTCONNECT SMTP server connection timeout in seconds
		CHRDS_SMTP_TIMEOUTSEND SMTP server send timeout in seconds
		CHRDS_SMTP_TLS Use TLS connection with SMTP server (true/false)
		CHRDS_SMTP_HOSTVERIFICATION Enable HostVerification by certificate for SMTP server connection (true/false)
		CHRDS_SMTP_FROM From address for SMTP server
		CHRDS_SMTP_QUEUE SMTP queue size
	*/

	log.Print("Loading configuration from alternating environment...")
	// The main API
	var httpHostENV string
	httpHostENV, _ = os.LookupEnv("CHRDS_HTTP_HOST")
	if httpHostENV != "" {
		appCfg.HTTP.HOST = httpHostENV
	}
	var httpPortENV string
	httpPortENV, _ = os.LookupEnv("CHRDS_HTTP_PORT")
	if httpPortENV != "" {
		appCfg.HTTP.PORT = httpPortENV
	}
	var httpReadTimeoutENV string
	httpReadTimeoutENV, _ = os.LookupEnv("CHRDS_HTTP_READTIMEOUT")
	if httpReadTimeoutENV != "" {
		var httpReadTimeoutINT int
		httpReadTimeoutINT, err = strconv.Atoi(httpReadTimeoutENV)
		if err != nil {
			log.Print("It was not possible to convert CHRDS_HTTP_READTIMEOUT (" + err.Error() + ")!")
		} else {
			appCfg.HTTP.READTIMEOUT = httpReadTimeoutINT
		}
	}
	var httpWriteTimeoutENV string
	httpWriteTimeoutENV, _ = os.LookupEnv("CHRDS_HTTP_READTIMEOUT")
	if httpWriteTimeoutENV != "" {
		var httpWriteTimeoutINT int
		httpWriteTimeoutINT, err = strconv.Atoi(httpWriteTimeoutENV)
		if err != nil {
			log.Print("It was not possible to convert CHRDS_HTTP_WRITETIMEOUT (" + err.Error() + ")!")
		} else {
			appCfg.HTTP.WRITETIMEOUT = httpWriteTimeoutINT
		}
	}
	var httpRLENV string
	httpRLENV, _ = os.LookupEnv("CHRDS_HTTP_RL")
	if httpRLENV != "" {
		var httpRLINT int
		httpRLINT, err = strconv.Atoi(httpRLENV)
		if err != nil {
			log.Print("It was not possible to convert CHRDS_HTTP_RL (" + err.Error() + ")!")
		} else {
			appCfg.HTTP.RL = httpRLINT
		}
	}
	var httpTLSENV string
	httpTLSENV, _ = os.LookupEnv("CHRDS_HTTP_TLS")
	if httpTLSENV != "" {
		boolValue, err := strconv.ParseBool(httpTLSENV)
		if err != nil {
			log.Print("It was not possible to convert CHRDS_HTTP_TLS (" + err.Error() + ")!")
		} else {
			appCfg.HTTP.TLS = boolValue
		}
	}
	var httpCertPathENV string
	httpCertPathENV, _ = os.LookupEnv("CHRDS_HTTP_CERTPATH")
	if httpCertPathENV != "" {
		appCfg.HTTP.CERTPATH = httpCertPathENV
	}
	var httpKeyPathENV string
	httpKeyPathENV, _ = os.LookupEnv("CHRDS_HTTP_KEYPATH")
	if httpKeyPathENV != "" {
		appCfg.HTTP.KEYPATH = httpKeyPathENV
	}
	var httpCAPathENV string
	httpCAPathENV, _ = os.LookupEnv("CHRDS_HTTP_CAPATH")
	if httpCAPathENV != "" {
		appCfg.HTTP.CAPATH = httpCAPathENV
	}

	// Worker
	var workerPoolENV string
	workerPoolENV, _ = os.LookupEnv("CHRDS_WORKER_POOL")
	if workerPoolENV != "" {
		var workerPoolINT int
		workerPoolINT, err = strconv.Atoi(workerPoolENV)
		if err != nil {
			log.Print("It was not possible to convert CHRDS_WORKER_POOL (" + err.Error() + ")!")
		} else {
			appCfg.WORKER.POOL = workerPoolINT
		}
	}
	var workerACTimeENV string
	workerACTimeENV, _ = os.LookupEnv("CHRDS_WORKER_ACTIME")
	if workerACTimeENV != "" {
		var workerACTimeINT int
		workerACTimeINT, err = strconv.Atoi(workerACTimeENV)
		if err != nil {
			log.Print("It was not possible to convert CHRDS_WORKER_ACTIME (" + err.Error() + ")!")
		} else {
			appCfg.WORKER.ACTIME = workerACTimeINT
		}
	}

	// Connection to the database
	var dbHostsENV string
	dbHostsENV, _ = os.LookupEnv("CHRDS_DB_HOST")
	if dbHostsENV != "" {
		dbHostsENV = strings.TrimSpace(dbHostsENV)       // Удаление возможных пробелов
		appCfg.DB.HOSTS = strings.Split(dbHostsENV, ",") // Разбиваем строку по символу ","
	}
	var dbUserNameENV string
	dbUserNameENV, _ = os.LookupEnv("CHRDS_DB_USERNAME")
	if dbUserNameENV != "" {
		appCfg.DB.USERNAME = dbUserNameENV
	}
	var dbPasswordENV string
	dbPasswordENV, _ = os.LookupEnv("CHRDS_DB_PASSWORD")
	if dbPasswordENV != "" {
		appCfg.DB.PASSWORD = dbPasswordENV
	}
	var dbCertPathENV string
	dbCertPathENV, _ = os.LookupEnv("CHRDS_DB_CERTPATH")
	if dbCertPathENV != "" {
		appCfg.DB.CERTPATH = dbCertPathENV
	}
	var dbKeyPathENV string
	dbKeyPathENV, _ = os.LookupEnv("CHRDS_DB_KEYPATH")
	if dbKeyPathENV != "" {
		appCfg.DB.KEYPATH = dbKeyPathENV
	}
	var dbCAPathENV string
	dbCAPathENV, _ = os.LookupEnv("CHRDS_DB_CAPATH")
	if dbCAPathENV != "" {
		appCfg.DB.CAPATH = dbCAPathENV
	}
	var dbKeySpaceENV string
	dbKeySpaceENV, _ = os.LookupEnv("CHRDS_DB_KEYSPACE")
	if dbKeySpaceENV != "" {
		appCfg.DB.KEYSPACE = dbKeySpaceENV
	}
	var dbConsistensyENV string
	dbConsistensyENV, _ = os.LookupEnv("CHRDS_DB_CONSISTENCY")
	if dbConsistensyENV != "" {
		appCfg.DB.CONSISTENCY = dbConsistensyENV
	}
	var dbConsistensyReadENV string
	dbConsistensyReadENV, _ = os.LookupEnv("CHRDS_DB_CONSISTENCYREAD")
	if dbConsistensyReadENV != "" {
		appCfg.DB.CONSISTENCYREAD = dbConsistensyReadENV
	}
	var dbTimeoutENV string
	dbTimeoutENV, _ = os.LookupEnv("CHRDS_DB_TIMEOUT")
	if dbTimeoutENV != "" {
		var dbTimeoutINT int
		dbTimeoutINT, err = strconv.Atoi(dbTimeoutENV)
		if err != nil {
			log.Print("It was not possible to convert CHRDS_DB_TIMEOUT (" + err.Error() + ")!")
		} else {
			appCfg.DB.TIMEOUT = dbTimeoutINT
		}
	}
	var dbTLSENV string
	dbTLSENV, _ = os.LookupEnv("CHRDS_DB_TLS")
	if dbTLSENV != "" {
		boolValue, err := strconv.ParseBool(dbTLSENV)
		if err != nil {
			log.Print("It was not possible to convert CHRDS_DB_TLS (" + err.Error() + ")!")
		} else {
			appCfg.DB.TLS = boolValue
		}
	}
	var dbHOSTVERIFICATIONENV string
	dbHOSTVERIFICATIONENV, _ = os.LookupEnv("CHRDS_DB_HOSTVERIFICATION")
	if dbHOSTVERIFICATIONENV != "" {
		boolValue, err := strconv.ParseBool(dbHOSTVERIFICATIONENV)
		if err != nil {
			log.Print("It was not possible to convert CHRDS_DB_HOSTVERIFICATION (" + err.Error() + ")!")
		} else {
			appCfg.DB.HOSTVERIFICATION = boolValue
		}
	}

	// ZBX server
	var zbxHostENV string
	zbxHostENV, _ = os.LookupEnv("CHRDS_ZBX_HOST")
	if zbxHostENV != "" {
		appCfg.ZBX.HOST = zbxHostENV
	}
	var zbxPortENV string
	zbxPortENV, _ = os.LookupEnv("CHRDS_ZBX_PORT")
	if zbxPortENV != "" {
		appCfg.ZBX.PORT = zbxPortENV
	}
	var zbxTypeENV string
	zbxTypeENV, _ = os.LookupEnv("CHRDS_ZBX_TYPE")
	if zbxTypeENV != "" {
		appCfg.ZBX.TYPE = zbxTypeENV
	}

	// ZBX Client
	var zbxPassivePortENV string
	zbxPassivePortENV, _ = os.LookupEnv("CHRDS_ZBX_PASSIVEPORT")
	if zbxPassivePortENV != "" {
		appCfg.ZBX.PASSIVEPORT = zbxPassivePortENV
	}
	var zbxPassiveTypeENV string
	zbxPassiveTypeENV, _ = os.LookupEnv("CHRDS_ZBX_PASSIVETYPE")
	if zbxPassiveTypeENV != "" {
		appCfg.ZBX.PASSIVETYPE = zbxPassiveTypeENV
	}

	// Swagger
	var swgTitleENV string
	swgTitleENV, _ = os.LookupEnv("CHRDS_SWG_TITLE")
	if swgTitleENV != "" {
		appCfg.SWG.TITLE = swgTitleENV
	}
	var swgDescriptionENV string
	swgDescriptionENV, _ = os.LookupEnv("CHRDS_SWG_DESCRIPTION")
	if swgDescriptionENV != "" {
		appCfg.SWG.DESCRIPTION = swgDescriptionENV
	}
	var swgVersionENV string
	swgVersionENV, _ = os.LookupEnv("CHRDS_SWG_VERSION")
	if swgVersionENV != "" {
		appCfg.SWG.VERSION = swgVersionENV
	}
	var swgHostENV string
	swgHostENV, _ = os.LookupEnv("CHRDS_SWG_HOST")
	if swgHostENV != "" {
		appCfg.SWG.HOST = swgHostENV
	}
	var swgBasePathENV string
	swgBasePathENV, _ = os.LookupEnv("CHRDS_SWG_BASEPATH")
	if swgBasePathENV != "" {
		appCfg.SWG.BASEPATH = swgBasePathENV
	}
	var swgSchemeENV string
	swgSchemeENV, _ = os.LookupEnv("CHRDS_SWG_SCHEME")
	if swgSchemeENV != "" {
		appCfg.SWG.SCHEME = swgSchemeENV
	}

	// SMTP server
	var smtpEnabledENV string
	smtpEnabledENV, _ = os.LookupEnv("CHRDS_SMTP_ENABLED")
	if smtpEnabledENV != "" {
		boolValue, err := strconv.ParseBool(smtpEnabledENV)
		if err != nil {
			log.Print("It was not possible to convert CHRDS_SMTP_ENABLED (" + err.Error() + ")!")
		} else {
			appCfg.SMTP.ENABLED = boolValue
		}
	}
	var smtpHostENV string
	smtpHostENV, _ = os.LookupEnv("CHRDS_SMTP_HOST")
	if smtpHostENV != "" {
		appCfg.SMTP.HOST = smtpHostENV
	}
	var smtpPortENV string
	smtpPortENV, _ = os.LookupEnv("CHRDS_SMTP_PORT")
	if smtpPortENV != "" {
		var smtpPortINT int
		smtpPortINT, err = strconv.Atoi(smtpPortENV)
		if err != nil {
			log.Print("It was not possible to convert CHRDS_SMTP_PORT (" + err.Error() + ")!")
		} else {
			appCfg.SMTP.PORT = smtpPortINT
		}
	}
	var smtpUserNameENV string
	smtpUserNameENV, _ = os.LookupEnv("CHRDS_SMTP_USERNAME")
	if smtpUserNameENV != "" {
		appCfg.SMTP.USERNAME = smtpUserNameENV
	}
	var smtpPasswordENV string
	smtpPasswordENV, _ = os.LookupEnv("CHRDS_SMTP_PASSWORD")
	if smtpPasswordENV != "" {
		appCfg.SMTP.PASSWORD = smtpPasswordENV
	}
	var smtpKeepAliveENV string
	smtpKeepAliveENV, _ = os.LookupEnv("CHRDS_SMTP_KEEPALIVE")
	if smtpKeepAliveENV != "" {
		boolValue, err := strconv.ParseBool(smtpKeepAliveENV)
		if err != nil {
			log.Print("It was not possible to convert CHRDS_SMTP_KEEPALIVE (" + err.Error() + ")!")
		} else {
			appCfg.SMTP.KEEPALIVE = boolValue
		}
	}
	var smtpTimeoutConnectENV string
	smtpTimeoutConnectENV, _ = os.LookupEnv("CHRDS_SMTP_TIMEOUTCONNECT")
	if smtpTimeoutConnectENV != "" {
		var smtpTimeoutConnectINT int
		smtpTimeoutConnectINT, err = strconv.Atoi(smtpTimeoutConnectENV)
		if err != nil {
			log.Print("It was not possible to convert CHRDS_SMTP_TIMEOUTCONNECT (" + err.Error() + ")!")
		} else {
			appCfg.SMTP.TIMEOUTCONNECT = smtpTimeoutConnectINT
		}
	}
	var smtpTimeoutSendENV string
	smtpTimeoutSendENV, _ = os.LookupEnv("CHRDS_SMTP_TIMEOUTSEND")
	if smtpTimeoutSendENV != "" {
		var smtpTimeoutSendINT int
		smtpTimeoutSendINT, err = strconv.Atoi(smtpTimeoutSendENV)
		if err != nil {
			log.Print("It was not possible to convert CHRDS_SMTP_TIMEOUTSEND (" + err.Error() + ")!")
		} else {
			appCfg.SMTP.TIMEOUTSEND = smtpTimeoutSendINT
		}
	}
	var smtpTLSEnv string
	smtpTLSEnv, _ = os.LookupEnv("CHRDS_SMTP_TLS")
	if smtpTLSEnv != "" {
		boolValue, err := strconv.ParseBool(smtpTLSEnv)
		if err != nil {
			log.Print("It was not possible to convert CHRDS_SMTP_TLS (" + err.Error() + ")!")
		} else {
			appCfg.SMTP.TLS = boolValue
		}
	}
	var smtpHostVerificationENV string
	smtpHostVerificationENV, _ = os.LookupEnv("CHRDS_SMTP_HOSTVERIFICATION")
	if smtpHostVerificationENV != "" {
		boolValue, err := strconv.ParseBool(smtpHostVerificationENV)
		if err != nil {
			log.Print("It was not possible to convert CHRDS_SMTP_HOSTVERIFICATION (" + err.Error() + ")!")
		} else {
			appCfg.SMTP.HOSTVERIFICATION = boolValue
		}
	}
	var smtpFromENV string
	smtpFromENV, _ = os.LookupEnv("CHRDS_SMTP_FROM")
	if smtpFromENV != "" {
		appCfg.SMTP.FROM = smtpFromENV
	}
	var smtpQueueENV string
	smtpQueueENV, _ = os.LookupEnv("CHRDS_SMTP_QUEUE")
	if smtpQueueENV != "" {
		var smtpQueueINT int
		smtpQueueINT, err = strconv.Atoi(smtpQueueENV)
		if err != nil {
			log.Print("It was not possible to convert CHRDS_SMTP_QUEUE (" + err.Error() + ")!")
		} else {
			appCfg.SMTP.QUEUE = smtpQueueINT
		}
	}

	log.Print("Configuration parameters:")
	log.Print("HTTP.HOST: ", appCfg.HTTP.HOST)
	log.Print("HTTP.PORT: ", appCfg.HTTP.PORT)
	log.Print("HTTP.READTIMEOUT: ", appCfg.HTTP.READTIMEOUT)
	log.Print("HTTP.WRITETIMEOUT: ", appCfg.HTTP.WRITETIMEOUT)
	log.Print("HTTP.RL: ", appCfg.HTTP.RL)
	log.Print("HTTP.TLS: ", appCfg.HTTP.TLS)
	log.Print("HTTP.CERTPATH: ", appCfg.HTTP.CERTPATH)
	log.Print("HTTP.KEYPATH: ", appCfg.HTTP.KEYPATH)
	log.Print("HTTP.CAPATH: ", appCfg.HTTP.CAPATH)

	log.Print("WORKER.POOL: ", appCfg.WORKER.POOL)
	log.Print("WORKER.POOL: ", appCfg.WORKER.ACTIME)

	log.Print("DB.HOSTS: ", appCfg.DB.HOSTS)
	log.Print("DB.USERNAME: ", appCfg.DB.USERNAME)
	log.Print("DB.TLS: ", appCfg.DB.TLS)
	log.Print("DB.CERTPATH: ", appCfg.DB.CERTPATH)
	log.Print("DB.KEYPATH: ", appCfg.DB.KEYPATH)
	log.Print("DB.CAPATH: ", appCfg.DB.CAPATH)
	log.Print("DB.HOSTVERIFICATION: ", appCfg.DB.HOSTVERIFICATION)
	log.Print("DB.KEYSPACE: ", appCfg.DB.KEYSPACE)
	log.Print("DB.CONSISTENCY: ", appCfg.DB.CONSISTENCY)
	log.Print("DB.CONSISTENCYREAD: ", appCfg.DB.CONSISTENCYREAD)
	log.Print("DB.TIMEOUT: ", appCfg.DB.TIMEOUT)

	log.Print("ZBX.HOST: ", appCfg.ZBX.HOST)
	log.Print("ZBX.PORT: ", appCfg.ZBX.PORT)
	log.Print("ZBX.TYPE: ", appCfg.ZBX.TYPE)
	log.Print("ZBX.PASSIVEPORT: ", appCfg.ZBX.PASSIVEPORT)
	log.Print("ZBX.PASSIVETYPE: ", appCfg.ZBX.PASSIVETYPE)

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
	log.Print("SMTP.HOSTVERIFICATION: ", appCfg.SMTP.HOSTVERIFICATION)
	log.Print("SMTP.FROM: ", appCfg.SMTP.FROM)
	log.Print("SMTP.QUEUE: ", appCfg.SMTP.QUEUE)

	return nil
}
