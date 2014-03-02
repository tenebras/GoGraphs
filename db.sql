-- DROP TABLE comment;
CREATE TABLE comment (
  comment_id bigserial NOT NULL,
  graph_id bigint,
  object_id bigint,
  ts timestamp without time zone,
  msg text,
  CONSTRAINT comment_pkey PRIMARY KEY (comment_id)
) WITH (OIDS=FALSE);

-- DROP INDEX comment_graph_id_and_ts;
CREATE INDEX comment_ts ON comment USING btree (ts);
CREATE INDEX comment_graph_id_and_ts ON comment USING btree (graph_id, ts);
CREATE INDEX comment_graph_id_and_ts_and_object_id ON comment USING btree (graph_id, ts, object_id);

-- DROP TABLE data;
CREATE TABLE data (
  data_id bigserial NOT NULL,
  graph_id bigint,
  ts timestamp without time zone,
  value double precision,
  object_id bigint,
  CONSTRAINT data_pkey PRIMARY KEY (data_id)
) WITH (OIDS=FALSE);

-- DROP INDEX data_graph_id_and_ts;
CREATE INDEX data_graph_id_and_ts ON data USING btree (graph_id, ts);
CREATE INDEX data_graph_id_and_ts_and_object_id ON data using btree (graph_id, ts, object_id);

CREATE TABLE meta (
	meta_id BIGSERIAL NOT NULL, 
	graph_id bigint,
	ts timestamp without time zone,
	value text,
	object_id double precision,
	CONSTRAINT meta_pkey PRIMARY KEY (meta_id)
);

CREATE INDEX meta_graph_id_and_ts ON data USING btree (graph_id, ts);
CREATE INDEX meta_graph_id_and_ts_and_object_id ON data USING btree (graph_id, ts, object_id);

-- DROP TABLE graph;
CREATE TABLE graph (
  graph_id bigserial NOT NULL,
  title character varying(255),
  added_at timestamp without time zone,
  updated_at timestamp without time zone,
  CONSTRAINT graph_pkey PRIMARY KEY (graph_id)
) WITH ( OIDS=FALSE );

-- DROP INDEX graph_updated_at;
CREATE INDEX graph_updated_at ON graph USING btree (updated_at);

CREATE TABLE collection(
	collection_id BIGSERIAL NOT NULL,
	title varchar(255),
	added_at timestamp without time zone,
	updates_at timestamp without time zone,
	structure text,
	CONSTRAINT collection_pkey PRIMARY KEY (collection_id)
);

CREATE INDEX collection_title ON collection USING btree (title);