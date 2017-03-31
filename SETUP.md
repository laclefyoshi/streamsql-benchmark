
# セットアップ手順 #

## Kafka ##

```
$ docker run -d -p 2181:2181 -p 9092:9092 --name kafka memsql/kafka
$ docker exec -it kafka /bin/bash
$ cd /opt/kafka_2.11-0.10.0.1/
$ ./bin/kafka-topics.sh --topic test --zookeeper 127.0.0.1:2181 --create --partitions 4 --replication-factor 1
$ ./bin/kafka-console-producer.sh --topic test --broker-list 127.0.0.1:9092
the quick
brown fox
jumped over
the lazy dog
```

## Memsql ##

サーバコンソール上で

```
$ docker run -d -p 3306:3306 -p 8080:9000 --name memsql memsql/quickstart
```

firewallに in-tcp:8080 を追加許可

http://サーバIP:8080/cluster をSafariで開けることを確認

```
$ docker exec -it memsql memsql
> CREATE DATABASE test_kafka;
> use test_kafka;
> CREATE TABLE messages (id text);
> CREATE PIPELINE `test_kafka` AS LOAD DATA KAFKA '172.17.0.3/test' INTO TABLE `messages`;
> TEST PIPELINE test_kafka LIMIT 1;
> START PIPELINE test_kafka;
> SELECT * FROM messages;
```

## PipelineDB ##

```
$ sudo mkdir /var/pipeline
$ docker run -d -p 5432:5432 -v /dev/shm:/dev/shm -v /var/pipeline:/mnt/pipelinedb --name pipelinedb pipelinedb/pipelinedb
$ sudo vi /var/pipelinedb/
shared_preload_libraries = 'cstore_fdw, postgres_fdw, pipeline_kafka'
$ docker stop pipelinedb
$ docker rm pipelinedb
$ docker run -d -p 5432:5432 -v /dev/shm:/dev/shm -v /var/pipeline:/mnt/pipelinedb --name pipelinedb pipelinedb/pipelinedb
```

```
$ docker exec -it pipelinedb bash
root@d0c5211cb62b:/# su - pipeline
pipeline@d0c5211cb62b:~$ psql -h 172.17.0.5 -p 5432 -d pipeline
Password: 
psql (9.5.3)
Type "help" for help.
# CREATE EXTENSION pipeline_kafka;
# SELECT pipeline_kafka.add_broker('172.17.0.3:9092');
# CREATE STREAM messages (id text);
# SELECT pipeline_kafka.consume_begin('test', 'messages', format := 'text');
# CREATE CONTINUOUS VIEW messages_view AS SELECT id FROM messages;
# SELECT * FROM messages_view;
```

## VoltDB ##

```
$ docker run -d -P -e HOST_COUNT=1 -e HOSTS=127.0.0.1 --name=voltdb voltdb/voltdb-community
```

```
$ docker exec -it voltdb bash
$ ./bin/sqlcmd
> CREATE TABLE messages (id varchar(128));
$ cd lib/
$ wget http://central.maven.org/maven2/org/apache/zookeeper/zookeeper/3.3.4/zookeeper-3.3.4.jar
$ wget http://central.maven.org/maven2/com/101tec/zkclient/0.3/zkclient-0.3.jar
$ ./bin/kafkaloader --zookeeper 172.17.0.3:2181 --topic test messages  # プロセス常駐
$ ./bin/sqlcmd
> SELECT * FROM messages;

```

## ここまでまとめ ##

```
$ docker ps
CONTAINER ID        IMAGE                     COMMAND                  CREATED             STATUS              PORTS                                                                                                                                                                                                                             NAMES
dca4a03192b0        pipelinedb/pipelinedb     "/scripts/startpipeli"   24 hours ago        Up 24 hours         0.0.0.0:5432->5432/tcp                                                                                                                                                                                                            pipelinedb
8416935f5ab0        voltdb/voltdb-community   "./docker-entrypoint."   25 hours ago        Up 25 hours         0.0.0.0:32785->22/tcp, 0.0.0.0:32784->3021/tcp, 0.0.0.0:32783->5555/tcp, 0.0.0.0:32782->7181/tcp, 0.0.0.0:32781->8080/tcp, 0.0.0.0:32780->8081/tcp, 0.0.0.0:32779->9000/tcp, 0.0.0.0:32778->21211/tcp, 0.0.0.0:32777->21212/tcp   voltdb
cf71e7f9b62b        memsql/kafka              "/scripts/start.sh"      28 hours ago        Up 28 hours         0.0.0.0:2181->2181/tcp, 0.0.0.0:9092->9092/tcp                                                                                                                                                                                    kafka
34b175444f06        memsql/quickstart         "/memsql-entrypoint.s"   29 hours ago        Up 29 hours         0.0.0.0:3306->3306/tcp, 0.0.0.0:8080->9000/tcp                                                                                                                                                                                    memsql
```

# データの投入 #

## Kafka Producerの起動 ##

Bitcoin情報とGithub履歴を投入する

```
$ docker cp kafka-producer kafka:/
$ docker exec -it kafka bash

$ cd /opt/kafka_2.11-0.10.0.1/
$ ./bin/kafka-topics.sh --topic test-bitcoin-j --zookeeper 127.0.0.1:2181 --create --partitions 4 --replication-factor 1
$ ./bin/kafka-topics.sh --topic test-geo-j --zookeeper 127.0.0.1:2181 --create --partitions 4 --replication-factor 1
$ cd /
$ nohup ./kafka-producer -topic test-bitcoin-j -url 'https://api.bitcoinaverage.com/ticker/global/all' -wait 30 &
$ nohup ./kafka-producer -topic test-geo-j -url 'https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/all_hour.geojson' -wait 20 &
```

```
$ docker exec -it memsql memsql
> use test_kafka;

> CREATE TABLE bitcoins (info json);
> CREATE PIPELINE `test_kafka_bitcoin` AS LOAD DATA KAFKA '172.17.0.3/test-bitcoin-j' INTO TABLE `bitcoins`;
> TEST PIPELINE test_kafka_bitcoin LIMIT 1;
> START PIPELINE test_kafka_bitcoin;
> SELECT * FROM bitcoins;

> CREATE TABLE geo (info json);
> CREATE PIPELINE `test_kafka_geo` AS LOAD DATA KAFKA '172.17.0.3/test-geo-j' INTO TABLE `geo`;
> TEST PIPELINE test_kafka_geo LIMIT 1;
> START PIPELINE test_kafka_geo;
> SELECT * FROM geo;
```

```
$ docker exec -it pipelinedb bash
root@d0c5211cb62b:/# su - pipeline
pipeline@d0c5211cb62b:~$ psql -h 172.17.0.5 -p 5432 -d pipeline
Password:

# CREATE STREAM bitcoins (info json);
# SELECT pipeline_kafka.consume_begin('test-bitcoin-j', 'bitcoins', format := 'json');
# CREATE CONTINUOUS VIEW bitcoins_view AS SELECT info FROM bitcoins;
# SELECT * FROM bitcoins_view;

# CREATE STREAM geo (info json);
# SELECT pipeline_kafka.consume_begin('test-geo-j', 'geo', format := 'json');
# CREATE CONTINUOUS VIEW geo_view AS SELECT info FROM geo;
# SELECT * FROM geo_view;
```


