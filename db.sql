-- Table: comment
-- DROP TABLE comment;
CREATE TABLE comment (
  comment_id bigserial NOT NULL,
  graph_id bigint,
  ts timestamp without time zone,
  msg text,
  CONSTRAINT comment_pkey PRIMARY KEY (comment_id)
) WITH (OIDS=FALSE);

ALTER TABLE comment OWNER TO postgres;

-- Index: comment_graph_id_and_ts
-- DROP INDEX comment_graph_id_and_ts;
CREATE INDEX comment_graph_id_and_ts ON comment USING btree (graph_id, ts);


-- Table: data
-- DROP TABLE data;
CREATE TABLE data (
  data_id bigserial NOT NULL,
  graph_id bigint,
  ts timestamp without time zone,
  value double precision,
  c1 double precision,
  c2 double precision,
  c3 double precision,
  params json,
  CONSTRAINT data_pkey PRIMARY KEY (data_id)
) WITH (OIDS=FALSE);

ALTER TABLE data OWNER TO postgres;

-- Index: data_graph_id_and_ts
-- DROP INDEX data_graph_id_and_ts;
CREATE INDEX data_graph_id_and_ts ON data USING btree (graph_id, ts);

-- Table: graph
-- DROP TABLE graph;

CREATE TABLE graph (
  graph_id bigserial NOT NULL,
  title character varying(64),
  added_at timestamp without time zone,
  updated_at timestamp without time zone,
  CONSTRAINT graph_pkey PRIMARY KEY (graph_id)
) WITH ( OIDS=FALSE );

ALTER TABLE graph OWNER TO postgres;

-- Index: graph_updated_at
-- DROP INDEX graph_updated_at;
CREATE INDEX graph_updated_at ON graph USING btree (updated_at);