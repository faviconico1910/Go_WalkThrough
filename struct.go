package main

// type SystemInfo struct {
// 	Name     string `json:"server_name"`
// 	CPU      string `json:"cpu_usage"`
// 	Ram      string `json:"ram_usage"`
// 	ErrorMsg string `json:"error_msg,omitempty"`
// }

// func main() {
// 	mySys := SystemInfo{
// 		Name: "MyServer",
// 		CPU:  "15%",
// 		Ram:  "60%",
// 	}
// 	// struct -> json
// 	jsonData, err := json.MarshalIndent(mySys, "", " ")
// 	if err != nil {
// 		fmt.Println("Error marshalling JSON:", err)
// 		return
// 	}

// 	fmt.Println(string(jsonData))

// 	// json -> struct
// 	rawJson := `{"server_name": "Database", "cpu_usage": "99.9%", "ram_usage": "85.0%"}`

// 	var parsedData SystemInfo
// 	err = json.Unmarshal([]byte(rawJson), &parsedData)
// 	if err != nil {
// 		fmt.Println("Error unmarshalling JSON:", err)
// 		return
// 	}
// 	fmt.Printf("Parsed Struct: %+v\n", parsedData)
// }
