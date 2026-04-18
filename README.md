# Charybdis Monitoring System DataManager 1.1.0

The **Charybdis Monitoring System** project is an attempt to create a simple infrastructure and application monitoring system based on Zabbix's best practices while addressing its main weaknesses in terms of scalability and data storage.  
**DataManager** is the server component responsible for collecting and storing data, as well as providing API functions for integrations. The key feature of **DataManager** is its compatibility with standard Zabbix agents.

## Update from version 1.0.4 or 1.0.5 to 1.1.0

- Use the update of the database scheme for version 1.1.0
- Update the configuration file, or variable environment
- Install version 1.1.0

## Installing DataManager

### Preparing the Environment
Download the latest release or clone the source code from the **master** branch:
```sh
git clone https://github.com/DemianSV/chrdsdatamanager.git
```

Install **Cassandra DB** using the instructions on the product's website (https://cassandra.apache.org/doc/stable/cassandra/installing/installing.html).  
>For demonstration purposes, you can use our Docker-based setup for a two-node cluster. The *docker-compose.yml* file is located in the *cassandra* directory.

Perform the initial database setup and load the schema from the *cassandra* directory (script *createDB.cql*):
```sh
# For Cassandra DB installed via Docker
docker exec -it cassandra01 cqlsh --username=username -f /root/createDB.cql
```

```sh
# For Cassandra DB installed directly on the server
cqlsh --username=username -f /path/to/createDB.cql
```

**Use your own credentials and script paths for the initial setup!**

### Installing DataManager via Docker Image
**Building and Running**  
Install **Docker** (if not already installed) using the latest instructions for your distribution (https://docs.docker.com/engine/install/).

Navigate to the directory containing the cloned repository.

Build the image (edit the *Dockerfile* if necessary):
```sh
docker build -t chrdsdatamanager
```

Configure environment variables and launch parameters in the *docker-compose.yml* file.

Start **DataManager** by running:
```sh
docker compose up -d
```

Verify the result:
```sh
docker ps -a
docker logs chrdsdatamanager01
```

**Stopping**  
```sh
docker compose down
```

### Installing DataManager Directly on the Server
**Building and Running**  
Install the latest version of GoLang from the product's website (https://go.dev).

Navigate to the directory containing the cloned repository.

Build **DataManager**:
```sh
go build
```

Configure the parameters in the *chrdsdatamanager.json* file.

Run **DataManager**:
```sh
./chrdsdatamanager
```

### Configuration Parameters
Configuration can be done in two ways:
1. Via the configuration file (*chrdsdatamanager.json*).
2. Via environment variables (recommended for Docker).

> **Environment variables have higher priority and will override parameters from the configuration file!**

> **Variable structure:**  
> **CHRDS**: Indicates the variable belongs to the monitoring system,  
> **HTTP**: Parameter group,  
> **HOST**: Configuration parameter.

**List of Supported Variables**

**CHRDS_DM_ID**: A unique identifier of the DataManager instance, required for precise distribution of tasks between instances (optional parameter).

**CHRDS_TLS_CERTPATH**: Path to the certificate for server connections,
**CHRDS_TLS_KEYPATH**: Path to the private key for server connections,
**CHRDS_TLS_KEYPASSWORD**: The password for the private key,
**CHRDS_TLS_CAPATH**: Path to the CA certificate for server connections.

**CHRDS_HTTP_HOST**: Address for incoming HTTP server connections (API, Swagger),  
**CHRDS_HTTP_PORT**: Port for incoming HTTP server connections,  
**CHRDS_HTTP_READTIMEOUT**: HTTP server Read TimeOut,  
**CHRDS_HTTP_WRITETIMEOUT**: HTTP server Write TimeOut,  
**CHRDS_HTTP_RL**: Requests Limit, HTTP server request rate limit per second,  
**CHRDS_HTTP_TLS**: Enable TLS for HTTP server (true/false),
**CHRDS_HTTP_TLS_SKIPVERIFY**: Do not check the TLS certificate.

**CHRDS_DB_HOST**: Array of database node addresses, comma-separated (e.g., 127.0.0.1:9042,127.0.0.1:9043),  
**CHRDS_DB_USERNAME**: Database username,  
**CHRDS_DB_PASSWORD**: Database password,  
**CHRDS_DB_TLS**: Enable TLS for database cluster connections (true/false),  
**CHRDS_DB_TTLS_SKIPVERIFY**: Do not check the TLS certificate,
**CHRDS_DB_KEYSPACE**: KeySpace for the connection,  
**CHRDS_DB_CONSISTENCY**: Consistency parameter for database writes,  
**CHRDS_DB_CONSISTENCYREAD**: Consistency parameter for database reads,  
**CHRDS_DB_TIMEOUT**: Query execution timeout (in milliseconds).
**CHRDS_BATCHSIZE**: 

**CHRDS_ZBX_HOST**: Address for incoming agent connections,  
**CHRDS_ZBX_PORT**: Port for incoming agent connections,  
**CHRDS_ZBX_TYPE**: Server protocol for agent communication (tcp),
**CHRDS_ZBX_PASSIVEPORT**: Agent port for server communication,
**CHRDS_ZBX_PASSIVETYPE**: Agent protocol for server communication (tcp),
**CHRDS_ZBX_TLS**: Enable TLS for zbx connections (true/false), 
**CHRDS_ZBX_TLS_SKIPVERIFY**: Do not check the TLS certificate,
**CHRDS_ZBX_WORKER_POOL**: Number of concurrent workers for active server-side checks,
**CHRDS_ZBX_WORKER_ACTIME**: Interval (in seconds) for running active server-side checks.

**CHRDS_PRMTEXPORTER_ENABLED**: Enable Prometheus exporter for DataManager,
**CHRDS_PRMTEXPORTER_METRICS_PATH** Set the path for publishing metrics.

**CHRDS_PRMTSCRAPER_ENABLED**: Enable Prometheus metrics polling feature,
**CHRDS_PRMTSCRAPER_INTERVAL**: Polling interval in seconds,
**CHRDS_PRMTSCRAPER_TIMEOUT**: Timeout while waiting for a response to a metrics request,
**CHRDS_PRMTSCRAPER_RETRYATTEMPTS**: Number of repeated requests if unsuccessful,
**CHRDS_PRMTSCRAPER_RETRYDELAY**:
**CHRDS_PRMTSCRAPER_MAXCONNECTIONS**:
**CHRDS_PRMTSCRAPER_WORKERSPOOL**: Pool size for concurrent requests,
**CHRDS_PRMTSCRAPER_TLS**: Enable TLS for prometheus connections (true/false),
**CHRDS_PRMTSCRAPER_TLS_SKIPVERIFY**: Do not check the TLS certificate.

**CHRDS_SWG_TITLE**: Swagger page title,  
**CHRDS_SWG_DESCRIPTION**: Swagger page description,  
**CHRDS_SWG_VERSION**: API version on Swagger,  
**CHRDS_SWG_HOST**: Full API address for Swagger,  
**CHRDS_SWG_BASEPATH**: Base API path for Swagger,  
**CHRDS_SWG_SCHEME**: Base protocol for Swagger.

**CHRDS_SMTP_ENABLED**: Enable the sending of warning on the SMTP protocol,
**CHRDS_SMTP_HOST**: SMTP server host,
**CHRDS_SMTP_PORT**: SMTP server port,
**CHRDS_SMTP_USERNAME**: SMTP user name,
**CHRDS_SMTP_PASSWORD**: Password,
**CHRDS_SMTP_KEEPALIVE**: Support Keepalive (true),
**CHRDS_SMTP_TIMEOUTCONNECT**: Timeout connection, seconds (10),
**CHRDS_SMTP_TIMEOUTSEND**:  Timeout sending, seconds (10),
**CHRDS_SMTP_TLS**: TLS support (true/false),
**CHRDS_SMTP_HOSTVERIFICATION**: The verification of the host (true/false),
**CHRDS_SMTP_FROM**: The address of the sender of the message,
**CHRDS_SMTP_QUEUE**: The size of the queue of messages (1000).

**Configuration via chrdsdatamanager.json File**  
> Parameter groups and names in the file are identical to those in the environment variables.

> When using the configuration file, remember that its parameters have lower priority and will be overridden by environment variables.

> Combining parameters from different sources is allowed, considering the above.

```json
{
	"DM": {
		"ID": ""
	},
	"TLS": {
		"CERTPATH": "server.crt",
		"KEYPATH": "server.key",
		"KEYPASSWORD": "",
		"CAPATH": "rootCA.crt"
	},
	"HTTP": {
		"HOST": "",
		"PORT": "6006",
		"READTIMEOUT": 10,
		"WRITETIMEOUT": 10,
		"RL": 100,
		"TLS": true,
		"SKIPVERIFY": true
	},
	"DB": {
		"HOSTS": ["127.0.0.1:9042", "127.0.0.1:9043"],
		"USERNAME": "chrds",
		"PASSWORD": "",
		"TLS": true,
		"SKIPVERIFY": true,
		"KEYSPACE": "chrds",
		"CONSISTENCY": "Quorum",
		"CONSISTENCYREAD": "One",
		"TIMEOUT": 10000,
		"BATCHSIZE": 300
	},
	"ZBX": {
		"HOST": "0.0.0.0",
		"PORT": "10051",
		"TYPE": "tcp",
		"PASSIVEPORT": "10050",
		"PASSIVETYPE": "tcp",
		"TLS": false,
		"SKIPVERIFY": true,
		"WORKERSPOOL": 3,
		"WORKERACTIME": 60
	},
	"PRMTSCRAPER": {
		"ENABLED": true,
		"INTERVAL": 60,
		"TIMEOUT": 10,
		"RETRYATTEMPTS": 3,
		"RETRYDELAY": 2,
		"MAXCONNECTIONS": 10,
		"WORKERSPOOL": 1,
		"TLS": false,
		"SKIPVERIFY": true
	},
	"PRMTEXPORTER": {
		"ENABLED": true,
		"METRICSPATH": "/metrics"
	},
	"SMTP": {
		"ENABLED": false,
		"HOST": "",
		"PORT": 587,
		"USERNAME": "",
		"PASSWORD": "",
		"KEEPALIVE": false,
		"TIMEOUTCONNECT": 10,
		"TIMEOUTSEND": 10,
		"TLS": true,
		"SKIPVERIFY": true,
		"FROM": "Monitoring",
		"QUEUE": 1000
	},
	"SWG": {
		"TITLE": "Swagger Charibdis DataManager API",
		"DESCRIPTION": "Monitoring system Charibdis DataManager",
		"VERSION": "1.0",
		"HOST": "your.domain:6443",
		"BASEPATH": "/api/v1",
		"SCHEME": "https"
	}
}
```

### Installing Zabbix Agent 2
Download Zabbix Agent 2 from the official Zabbix website (https://www.zabbix.com/download) or install it using your distribution's package manager.

### Configuring Zabbix Agent 2
Edit the configuration file `/etc/zabbix/zabbix_agent2.conf`.

```
##### Passive checks related

### Option: Server

# Address of DataManager. Only connections from this address will be accepted in passive mode.
Server=127.0.0.1
```

```
### Option: ListenPort

# Address on which Zabbix Agent 2 will listen.
ListenIP=127.0.0.1
```

```
##### Active checks related

### Option: ServerActive

# Address and port of DataManager, corresponding to CHRDS_ZBX_HOST and CHRDS_ZBX_PORT in DataManager.
ServerActive=127.0.0.1:10051

# Module ID assigned during agent registration in UIManager, under the "Modules" section.
Hostname=427fc5f7-7625-4a85-a12e-1349c6ac52a0
```

**Example of agent registration in UIManager, "Modules" section**

<img width="960" src="https://github.com/user-attachments/assets/7930f21b-15f7-477c-a3d4-e19f78c97068" />
