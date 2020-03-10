package main

import (
    "encoding/json"
    "os"
	"fmt"
	"path"
	"errors"
)

type JsonConfig struct {
	HOST         string
	PORT         string
	ImgPath      string `json:"IMG_PATH"`
	QUALITY      string
	AllowedTypes []string `json:"ALLOWED_TYPES"`
	ExhaustPath  string   `json:"EXHAUST_PATH"`
	EXTRA        []ExtraConfig
}

type Config struct {
	HOST         string
	PORT         string
	ImgPath      string `json:"IMG_PATH"`
	QUALITY      string
	AllowedTypes []string `json:"ALLOWED_TYPES"`
	ExhaustPath  string   `json:"EXHAUST_PATH"`
	EXTRA        map[string]ExtraConfig
}

type ExtraConfig struct {
	ServerName      string `json:"SERVER_NAME"`
	ImgPath         string `json:"IMG_PATH"`
	ExhaustPath     string `json:"EXHAUST_PATH"`
}

func LoadConfig(path string) (Config,error){
	var jconfig JsonConfig
	var config Config
	var err error
    filePtr, _ := os.Open(path)
    defer filePtr.Close()
 
    decoder := json.NewDecoder(filePtr)
 
    errs := decoder.Decode(&jconfig)
    if errs!=nil{
        err = errors.New("error:read config file fail")
	}
	config.HOST = jconfig.HOST
	config.PORT = jconfig.PORT
	config.QUALITY = jconfig.QUALITY
	config.AllowedTypes = jconfig.AllowedTypes
	config.ImgPath = jconfig.ImgPath
	config.ExhaustPath = jconfig.ExhaustPath

	if jconfig.EXTRA == nil {
		config.EXTRA = nil;
	}else{
		config.EXTRA = make(map[string]ExtraConfig)
		for _,value := range jconfig.EXTRA{
			config.EXTRA[value.ServerName] = value
		}
	}
	fmt.Println(config)
	return config,err
}


func (config *Config)GetExhaustPath(serverName string) (exhaustPath string,err error){
	if config.EXTRA == nil{
		if len(config.ExhaustPath) == 0{
			exhaustPath = "./exhaust"
		}else{
			exhaustPath = path.Clean(config.ExhaustPath)
		}
	}else{
		if value,exist := config.EXTRA[serverName]; exist{
			if len(value.ExhaustPath) == 0{
				exhaustPath = "./exhaust"
			}else{
				exhaustPath = path.Clean(value.ExhaustPath)
			}
		}else{
			err = errors.New("error: server name is not exist")
			exhaustPath = ""
			return 
		}
	}
	return 	
}

func (config *Config)GetImagePath(serverName string)  string{
	if config.EXTRA == nil{
		return config.ImgPath
	}else{
		return path.Clean(config.EXTRA[serverName].ImgPath)
	}
}

func (config *Config)GetAllImagePathAndExhaustPath() []ExtraConfig{
	all := make([]ExtraConfig,0)
	if config.EXTRA == nil{
		if len(config.ExhaustPath) == 0 {
			config.ExhaustPath = "./exhaust"
		}
		single :=ExtraConfig{
			ImgPath : config.ImgPath,
			ExhaustPath : config.ExhaustPath,
		}
		all = append(all,single)
	}else{
		for _, v := range config.EXTRA {
			if len(v.ExhaustPath) == 0 {
				v.ExhaustPath = "./exhaust"
			}
			single :=ExtraConfig{
				ImgPath : v.ImgPath,
				ExhaustPath : v.ExhaustPath,
			}
			all = append(all,single)
		}
	}
	return all
}

// func main(){
// 	config,_ := LoadConfig("./config-example.json")
// 	// // for k,v := range config.GetAllImagePathAndExhaustPath(){
// 	// // 	fmt.Println(k)
// 	// // 	fmt.Println(v.ImgPath)
// 	// // 	fmt.Println(v.ExhaustPath)
// 	// // }
// 	fmt.Println(config.GetExhaustPath("webptest2.keshane.moe"))
// }