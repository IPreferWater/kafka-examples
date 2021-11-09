package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/riferrei/srclient"
)

var (
	kafkaUrl          string
	schemaRegistryUrl string
	securityProtocol  string

	topic          string
	storeID        int
	inventoryID    int
	countEpcToSend int
)

type PikmonValue struct {
	Color string `json:"color"`
	Type  string `json:"type"`
	State string `json:"state"`
}

type PikmonKey struct {
	ID      int    `json:"id"`
	Captain string `json:"captain"`
}

func main() {
	initVarsFromEnd()
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGKILL)

	go Start()
	go Read()

	sig := <-signals
	fmt.Println("Got signal: ", sig)

}
func getKafkaConfigMap() *kafka.ConfigMap {
	c := &kafka.ConfigMap{
		"bootstrap.servers": kafkaUrl,
		"security.protocol": securityProtocol,
	}

	if securityProtocol != "PLAINTEXT" {
		c.SetKey("ssl.ca.location", "/go-producer/certs/ca.pem")
		c.SetKey("ssl.certificate.location", "/go-producer/certs/service.cert")
		c.SetKey("ssl.key.location", "/go-producer/certs/service.key")
	}

	return c
}
func Start() {

	kafkaConfig := getKafkaConfigMap()
	p, err := kafka.NewProducer(kafkaConfig)
	if err != nil {
		panic(fmt.Sprintf("error creating producer %s", err))
	}
	defer p.Close()

	// this will check the status of the sent messages
	go func() {
		for event := range p.Events() {
			switch ev := event.(type) {
			case *kafka.Message:
				m := ev
				if ev.TopicPartition.Error != nil {
					fmt.Printf("Error delivering the message '%s'\n error %s\n", m.Key, ev.TopicPartition.Error)
				}
				// uncoment this part to debug your succes
				/*else {
					//fmt.Printf("Message delivered successfully!\n key : %s\n headers : %s\n opaque : %s\n timestamp: %s\n value : %s \n offet : %s \n partition : %d\n topic : %s\n", m.Key, m.Headers, m.Opaque, m.Timestamp, m.Value, m.TopicPartition.Offset, m.TopicPartition.Partition, m.TopicPartition.Topic)
				}*/
			}
		}
	}()

	schemaRegistryClient := srclient.CreateSchemaRegistryClient(schemaRegistryUrl)

	schemaValue := getSchema(schemaRegistryClient, fmt.Sprintf("pikmin-value"))
	schemaKey := getSchema(schemaRegistryClient, fmt.Sprintf("pikmin-key"))

	for i := 1; i <= countEpcToSend; i++ {

		//value
		value := PikmonValue{
			Color: "blue",
			Type:  "flower",
			State: "waiting",
		}

		recordValue := getValueByte(schemaValue, value)

		//key
		key := PikmonKey{
			ID:      i,
			Captain: "olimar",
		}
		keyValue := getValueByte(schemaKey, key)

		//header
		header := []kafka.Header{
			{
				Key:   "planet",
				Value: []byte("blue 3"),
			},
		}

		errProduce := p.Produce(
			&kafka.Message{
				Key:            keyValue,
				Value:          recordValue,
				TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: -1},
				Headers:        header,
			}, nil)

		if errProduce != nil {
			panic(fmt.Sprintf("errProduce %s", errProduce))
		}

		if i%10000 == 0 {
			i := p.Flush(2 * 1000)
			fmt.Println(i)
		}

	}
	i := p.Flush(2 * 1000)
	fmt.Println(i)
}

func Read() {
	kafkaConfig := getKafkaConfigMap()
	kafkaConfig.SetKey("group.id", "go-app")
	kafkaConfig.SetKey("auto.offset.reset", "earliest")
	consumer, err := kafka.NewConsumer(kafkaConfig)

	if err != nil {
		panic(fmt.Sprintf("error creating consumer %s", err))
	}

	err = consumer.SubscribeTopics([]string{topic}, nil)

	schemaRegistryClient := srclient.CreateSchemaRegistryClient(schemaRegistryUrl)

	fmt.Println("start read")
	for {
		msg, err := consumer.ReadMessage(-1)
		if err != nil {
			panic(fmt.Sprintf("error readin msg %s\n", err))
		}
		// 3) Recover the schema id from the message and use the
		// client to retrieve the schema from Schema Registry.
		// Then use it to deserialize the record accordingly.
		schemaID := binary.BigEndian.Uint32(msg.Value[1:5])
		fmt.Printf("schema ID = %d\n", schemaID)

		schema, err := schemaRegistryClient.GetSchema(int(schemaID))

		if err != nil {
			panic(fmt.Sprintf("Error getting the schema with id '%d' %s", schemaID, err))
		}

		native, _, _ := schema.Codec().NativeFromBinary(msg.Value[5:])
		value, _ := schema.Codec().TextualFromNative(nil, native)
		fmt.Printf("Here is the message %s\n", string(value))
	}
}

func initVarsFromEnd() {
	kafkaUrl = os.Getenv("KAFKA_URL")
	schemaRegistryUrl = os.Getenv("SCHEMA_REGISTRY_URL")
	securityProtocol = os.Getenv("SECURITY_PROTOCOL")

	topic = os.Getenv("TOPIC")
	storeID = getIntFromEnv("STORE_ID")
	inventoryID = getIntFromEnv("INVENTORY_ID")
	countEpcToSend = getIntFromEnv("COUNT_EPC_TO_SEND")
}

func getIntFromEnv(envStr string) int {
	intValue, err := strconv.Atoi(os.Getenv(envStr))
	if err != nil {
		panic(fmt.Sprintf("error trying to parse int envValue %s => %s\n", envStr, err))
	}

	return intValue
}
