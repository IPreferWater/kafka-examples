package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/riferrei/srclient"
)

func getSchema(client *srclient.SchemaRegistryClient, schemaName string) *srclient.Schema {
	schema, err := client.GetLatestSchema(schemaName)
	if err != nil {
		panic(fmt.Sprintf("error getLatestSchema %s", err))
	}

	if schema == nil {
		panic(fmt.Sprintf("there is no schema with name %s\n", schemaName))
	}

	return schema
}

func getValueByte(schema *srclient.Schema, toMarshall interface{}) []byte {

	schemaIDBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(schemaIDBytes, uint32(schema.ID()))

	value, errJ := json.Marshal(toMarshall)
	if errJ != nil {
		panic(fmt.Sprintf("err jsonMarshall %s", errJ))
	}

	native, _, err := schema.Codec().NativeFromTextual(value)
	if err != nil {
		panic(fmt.Sprintf("err NativeFromTextual %s", err))
	}
	valueBytes, err := schema.Codec().BinaryFromNative(nil, native)

	if err != nil {
		panic(fmt.Sprintf("err BinaryFromNative %s", err))
	}

	var recordValue []byte
	recordValue = append(recordValue, byte(0))
	recordValue = append(recordValue, schemaIDBytes...)
	recordValue = append(recordValue, valueBytes...)

	return recordValue
}
