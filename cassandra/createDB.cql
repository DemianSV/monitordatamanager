/* Version 1.0.4 INIT */

/* Create the space of the Chrds keys */
CREATE KEYSPACE IF NOT EXISTS chrds WITH replication = {'class': 'NetworkTopologyStrategy', 'datacenter01': '1'}  AND durable_writes = true;

/* Create a user chrds (Password: CHRDSpass5) for applications */
CREATE ROLE IF NOT EXISTS chrds with SUPERUSER = false AND LOGIN = true and PASSWORD = 'CHRDSpass5';

/* We give out the necessary grants */
GRANT ALL ON KEYSPACE chrds TO chrds;
GRANT EXECUTE ON ALL FUNCTIONS TO chrds;
GRANT SELECT ON system.peers TO chrds; /* Get data about the node connection */
GRANT SELECT ON system.local TO chrds; /* Get data about cluster nodes */

/* We create a tabular structure */
CREATE TABLE IF NOT EXISTS chrds.dashboard (
    user_id uuid,
    id uuid,
    name text,
    start_time bigint,
    status int,
    stop_time bigint,
    PRIMARY KEY (user_id, id)
) WITH CLUSTERING ORDER BY (id ASC);

CREATE TABLE IF NOT EXISTS chrds.dashboard_chartdata (
    dashboard_id uuid,
    graph_order int,
    id uuid,
    graph_color text,
    graph_func text,
    graph_type text,
    metric text,
    space_id uuid,
    status int,
    PRIMARY KEY (dashboard_id, graph_order, id)
) WITH CLUSTERING ORDER BY (graph_order ASC, id ASC);

CREATE TABLE IF NOT EXISTS chrds.owner_id_user (
    owner_id uuid,
    user_name text,
    PRIMARY KEY (owner_id, user_name)
) WITH CLUSTERING ORDER BY (user_name ASC);

CREATE TABLE IF NOT EXISTS chrds.problem (
    space_id uuid,
    module_id uuid,
    metric text,
    event_time bigint,
    event_time_start bigint,
    status text,
    value text,
    PRIMARY KEY (space_id, module_id, metric)
) WITH CLUSTERING ORDER BY (module_id ASC, metric ASC);

CREATE TABLE IF NOT EXISTS chrds.raw_aggregation (
    space_id uuid,
    object text,
    metric text,
    synt_key text,
    count int,
    max_value float,
    min_value float,
    sum_value float,
    PRIMARY KEY ((space_id, object, metric, synt_key))
);

CREATE TABLE IF NOT EXISTS chrds.raw_data01 (
    space_id uuid,
    synt_key text,
    event_time bigint,
    create_time bigint,
    metric text,
    object text,
    status int,
    value float,
    PRIMARY KEY ((space_id, synt_key), event_time)
) WITH CLUSTERING ORDER BY (event_time DESC);

CREATE TABLE IF NOT EXISTS chrds.raw_data02 (
    space_id uuid,
    metric text,
    synt_key text,
    event_time bigint,
    create_time bigint,
    object text,
    status int,
    value float,
    PRIMARY KEY ((space_id, metric, synt_key), event_time)
) WITH CLUSTERING ORDER BY (event_time DESC);

CREATE TABLE IF NOT EXISTS chrds.raw_data03 (
    space_id uuid,
    synt_key text,
    object text,
    event_time bigint,
    create_time bigint,
    metric text,
    status int,
    value float,
    PRIMARY KEY ((space_id, synt_key, object), event_time)
) WITH CLUSTERING ORDER BY (event_time DESC);

CREATE TABLE IF NOT EXISTS chrds.raw_key (
    space_id uuid,
    type_data text,
    flag int,
    synt_key text,
    PRIMARY KEY ((space_id, type_data, flag), synt_key)
) WITH CLUSTERING ORDER BY (synt_key DESC);

CREATE TABLE IF NOT EXISTS chrds.raw_text01 (
    space_id uuid,
    synt_key text,
    event_time bigint,
    create_time bigint,
    metric text,
    object text,
    status int,
    value text,
    PRIMARY KEY ((space_id, synt_key), event_time)
) WITH CLUSTERING ORDER BY (event_time DESC);

CREATE TABLE IF NOT EXISTS chrds.raw_text02 (
    space_id uuid,
    metric text,
    synt_key text,
    event_time bigint,
    create_time bigint,
    object text,
    status int,
    value text,
    PRIMARY KEY ((space_id, metric, synt_key), event_time)
) WITH CLUSTERING ORDER BY (event_time ASC);

CREATE TABLE IF NOT EXISTS chrds.registration (
    id uuid PRIMARY KEY,
    description text,
    status int,
    type text,
    user_id uuid
);

CREATE TABLE IF NOT EXISTS chrds.space (
    id uuid PRIMARY KEY,
    description text,
    status int,
    user_id uuid
);

CREATE TABLE IF NOT EXISTS chrds.space_metric_data (
    space_id uuid,
    metric text,
    PRIMARY KEY (space_id, metric)
) WITH CLUSTERING ORDER BY (metric ASC);

CREATE TABLE IF NOT EXISTS chrds.space_metric_text (
    space_id uuid,
    metric text,
    PRIMARY KEY (space_id, metric)
) WITH CLUSTERING ORDER BY (metric ASC);

CREATE TABLE IF NOT EXISTS chrds.task (
    module_id uuid,
    space_id uuid,
    object text,
    metric text,
    critical text,
    data_type text,
    int_id bigint,
    interval bigint,
    status int,
    warning text,
    PRIMARY KEY (module_id, space_id, object, metric)
) WITH CLUSTERING ORDER BY (space_id ASC, object ASC, metric ASC);

CREATE TABLE IF NOT EXISTS chrds.user (
    user_name text PRIMARY KEY,
    description text,
    first_name text,
    last_name text,
    login_time bigint,
    owner_id uuid,
    password text,
    role text,
    status int,
    user_id uuid
);

CREATE TABLE IF NOT EXISTS chrds.user_id (
    user_id uuid PRIMARY KEY,
    user_name text
);

CREATE TABLE IF NOT EXISTS chrds.user_registration (
    user_id uuid,
    registration_id uuid,
    PRIMARY KEY (user_id, registration_id)
) WITH CLUSTERING ORDER BY (registration_id ASC);

CREATE TABLE IF NOT EXISTS chrds.user_space (
    user_id uuid,
    space_id uuid,
    PRIMARY KEY (user_id, space_id)
) WITH CLUSTERING ORDER BY (space_id ASC);

/* We create an intended superpoler (username: superadmin, password: superadmin) */
INSERT INTO chrds.user ("user_name", "description", "first_name", "last_name", "password", "role", "status", "user_id") VALUES ('superadmin', 'Monitoring System Administrator', 'Admin', 'SuperAdmin', '186cf774c97b60a1c106ef718d10970a6a06e06bef89553d9ae65d938a886eae', 'superadmin', 1, 4467f7dd-9787-4113-8e4a-abdfd83956c6);
INSERT INTO chrds.user_id ("user_id", "user_name") VALUES (4467f7dd-9787-4113-8e4a-abdfd83956c6, 'superadmin');

/* We create a pre -designed space and module for the interaction of UIManager and DataManager */
INSERT INTO chrds.registration (id, type, status, description, user_id) VALUES (623a57c2-3df5-4287-ada5-82f7e4a0b5db, 'uimanager', 1, 'UIManager', 4467f7dd-9787-4113-8e4a-abdfd83956c6);
INSERT INTO chrds.user_registration (user_id, registration_id) VALUES (4467f7dd-9787-4113-8e4a-abdfd83956c6, 623a57c2-3df5-4287-ada5-82f7e4a0b5db);
INSERT INTO chrds.space (id, description, status, user_id) VALUES (17f0bd20-41cf-4801-a481-ff721a41fa93, 'UIManager', 1, 4467f7dd-9787-4113-8e4a-abdfd83956c6);
INSERT INTO chrds.user_space (user_id, space_id) VALUES (4467f7dd-9787-4113-8e4a-abdfd83956c6, 17f0bd20-41cf-4801-a481-ff721a41fa93);

/* Version 1.0.5 */

/* We add a new column space_description to all tables that contain space_id */
ALTER TABLE chrds.raw_data01 ADD IF NOT EXISTS space_description text;
ALTER TABLE chrds.raw_data02 ADD IF NOT EXISTS space_description text;
ALTER TABLE chrds.raw_data03 ADD IF NOT EXISTS space_description text;
ALTER TABLE chrds.raw_text01 ADD IF NOT EXISTS space_description text;
ALTER TABLE chrds.raw_text02 ADD IF NOT EXISTS space_description text;
ALTER TABLE chrds.raw_aggregation ADD IF NOT EXISTS space_description text;

CREATE TABLE IF NOT EXISTS chrds.emaillist (
    user_id uuid,
    id uuid,
    name text,
    description text,
    status int,
    PRIMARY KEY (user_id, id)
);

CREATE TABLE IF NOT EXISTS chrds.emaillist_data (
    emaillist_id uuid,
    id uuid,
    email text,
    description text,
    status int,
    PRIMARY KEY (emaillist_id, id)
);

ALTER TABLE chrds.task ADD IF NOT EXISTS emaillist_id uuid;

/* Version 1.1.0 */

/* Fields for supporting Prometheus protocol and optimistic blocking */
ALTER TABLE chrds.task ADD IF NOT EXISTS ol uuid;
ALTER TABLE chrds.task ADD IF NOT EXISTS ol_time bigint;

ALTER TABLE chrds.registration ADD IF NOT EXISTS address text;

ALTER TABLE chrds.raw_data01 ADD IF NOT EXISTS labels map<text, text>;
ALTER TABLE chrds.raw_data02 ADD IF NOT EXISTS labels map<text, text>;
ALTER TABLE chrds.raw_data03 ADD IF NOT EXISTS labels map<text, text>;

ALTER TABLE chrds.dashboard ADD IF NOT EXISTS interval bigint;

ALTER TABLE IF EXISTS raw_data01 DROP IF EXISTS status;
ALTER TABLE IF EXISTS raw_data02 DROP IF EXISTS status;
ALTER TABLE IF EXISTS raw_data03 DROP IF EXISTS status;
ALTER TABLE IF EXISTS raw_text01 DROP IF EXISTS status;
ALTER TABLE IF EXISTS raw_text02 DROP IF EXISTS status;

/* We set the compaction strategy and data expiration (TTL) for raw data tables */
ALTER TABLE chrds.raw_data01 WITH compaction = {'class': 'TimeWindowCompactionStrategy', 'compaction_window_size': '1', 'compaction_window_unit': 'DAYS'} AND default_time_to_live = 2592000;
ALTER TABLE chrds.raw_data02 WITH compaction = {'class': 'TimeWindowCompactionStrategy', 'compaction_window_size': '1', 'compaction_window_unit': 'DAYS'} AND default_time_to_live = 2592000;
ALTER TABLE chrds.raw_data03 WITH compaction = {'class': 'TimeWindowCompactionStrategy', 'compaction_window_size': '1', 'compaction_window_unit': 'DAYS'} AND default_time_to_live = 2592000;
ALTER TABLE chrds.raw_text01 WITH compaction = {'class': 'TimeWindowCompactionStrategy', 'compaction_window_size': '1', 'compaction_window_unit': 'DAYS'} AND default_time_to_live = 2592000;
ALTER TABLE chrds.raw_text02 WITH compaction = {'class': 'TimeWindowCompactionStrategy', 'compaction_window_size': '1', 'compaction_window_unit': 'DAYS'} AND default_time_to_live = 2592000;

ALTER TABLE chrds.task ADD IF NOT EXISTS dm_id uuid;

/* Оптимизация хранения метрик и текстовых данных */
CREATE TABLE chrds.raw_text (
    space_id uuid,
    metric text,
    event_time bigint,
    create_time bigint,
    object text,
    space_description text,
    value text,
    PRIMARY KEY ((space_id, metric), event_time)
) WITH CLUSTERING ORDER BY (event_time DESC)
    AND additional_write_policy = '99p'
    AND allow_auto_snapshot = true
    AND bloom_filter_fp_chance = 0.01
    AND caching = {'keys': 'ALL', 'rows_per_partition': 'NONE'}
    AND cdc = false
    AND comment = ''
    AND compaction = {'class': 'org.apache.cassandra.db.compaction.TimeWindowCompactionStrategy', 'compaction_window_size': '1', 'compaction_window_unit': 'DAYS', 'max_threshold': '32', 'min_threshold': '4'}
    AND compression = {'chunk_length_in_kb': '16', 'class': 'org.apache.cassandra.io.compress.LZ4Compressor'}
    AND memtable = 'default'
    AND crc_check_chance = 1.0
    AND default_time_to_live = 2592000
    AND extensions = {}
    AND gc_grace_seconds = 864000
    AND incremental_backups = true
    AND max_index_interval = 2048
    AND memtable_flush_period_in_ms = 0
    AND min_index_interval = 128
    AND read_repair = 'BLOCKING'
    AND speculative_retry = '99p';

CREATE TABLE chrds.raw_data (
    space_id uuid,
    metric text,
    event_time bigint,
    create_time bigint,
    object text,
    space_description text,
    value float,
    labels map<text, text>,
    PRIMARY KEY ((space_id, metric), event_time)
) WITH CLUSTERING ORDER BY (event_time DESC)
    AND additional_write_policy = '99p'
    AND allow_auto_snapshot = true
    AND bloom_filter_fp_chance = 0.01
    AND caching = {'keys': 'ALL', 'rows_per_partition': 'NONE'}
    AND cdc = false
    AND comment = ''
    AND compaction = {'class': 'org.apache.cassandra.db.compaction.TimeWindowCompactionStrategy', 'compaction_window_size': '1', 'compaction_window_unit': 'DAYS', 'max_threshold': '32', 'min_threshold': '4'}
    AND compression = {'chunk_length_in_kb': '16', 'class': 'org.apache.cassandra.io.compress.LZ4Compressor'}
    AND memtable = 'default'
    AND crc_check_chance = 1.0
    AND default_time_to_live = 2592000
    AND extensions = {}
    AND gc_grace_seconds = 864000
    AND incremental_backups = true
    AND max_index_interval = 2048
    AND memtable_flush_period_in_ms = 0
    AND min_index_interval = 128
    AND read_repair = 'BLOCKING'
    AND speculative_retry = '99p';

/* Счетчик записей raw таблиц */
CREATE TABLE chrds.raw_count (
    space_id uuid,
    raw_type text,
    value bigint,
    create_time bigint,
    PRIMARY KEY ((space_id, raw_type), create_time)
) WITH CLUSTERING ORDER BY (create_time DESC)
    AND additional_write_policy = '99p'
    AND allow_auto_snapshot = true
    AND bloom_filter_fp_chance = 0.01
    AND caching = {'keys': 'ALL', 'rows_per_partition': 'NONE'}
    AND cdc = false
    AND comment = ''
    AND compaction = {'class': 'org.apache.cassandra.db.compaction.TimeWindowCompactionStrategy', 'compaction_window_size': '1', 'compaction_window_unit': 'DAYS', 'max_threshold': '32', 'min_threshold': '4'}
    AND compression = {'chunk_length_in_kb': '16', 'class': 'org.apache.cassandra.io.compress.LZ4Compressor'}
    AND memtable = 'default'
    AND crc_check_chance = 1.0
    AND default_time_to_live = 2592000
    AND extensions = {}
    AND gc_grace_seconds = 864000
    AND incremental_backups = true
    AND max_index_interval = 2048
    AND memtable_flush_period_in_ms = 0
    AND min_index_interval = 128
    AND read_repair = 'BLOCKING'
    AND speculative_retry = '99p';

    DROP TABLE raw_data01;
    DROP TABLE raw_data02;
    DROP TABLE raw_data03;
    DROP TABLE raw_text01;
    DROP TABLE raw_text02;
    DROP TABLE raw_key;

    GRANT SELECT ON chrds.raw_data TO grafana;