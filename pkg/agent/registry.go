package agent

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type InputStruct struct {
	Name    string
	Message string
}

// 定义输出结构体
type OutputStruct struct {
	Greeting string
	Original InputStruct
}

// Hello 函数接收 InputStruct，返回 OutputStruct
func Hello(input InputStruct) OutputStruct {
	fmt.Println("hello") // 打印 hello

	// 创建并返回输出结构体
	return OutputStruct{
		Greeting: "Hello, " + input.Name,
		Original: input,
	}
}

type FunctionHandler func(*structpb.Struct) (*structpb.Struct, error)

var functionRegistry = map[string]FunctionHandler{
	"Hello": helloHandler,
}

func helloHandler(params *structpb.Struct) (*structpb.Struct, error) {
	var input InputStruct

	// Step 1: 将 protobuf Struct 参数转为 InputStruct
	paramJson, err := protojson.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	if err := json.Unmarshal(paramJson, &input); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	// Step 2: 调用逻辑函数
	output := Hello(input)

	// Step 3: 把输出结构体转为 JSON -> map[string]interface{} -> *structpb.Struct
	outputJson, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("output marshal to json error: %w", err)
	}
	var outputMap map[string]any
	if err := json.Unmarshal(outputJson, &outputMap); err != nil {
		return nil, fmt.Errorf("output unmarshal to map error: %w", err)
	}
	resultStruct, err := structpb.NewStruct(outputMap)
	if err != nil {
		return nil, fmt.Errorf("result marshal error: %w", err)
	}
	return resultStruct, nil
}
