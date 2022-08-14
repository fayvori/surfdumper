package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

type Country struct {
	Id        string      `json:"id"`
	Name      interface{}
	Code      string      `json:"code"`
	CountryId string      `json:"countryId"`
	Longitude string      `json:"longitude"`
	StateCode interface{} `json:"state_code"`
	Latitude  string      `json:"latitude"`
}

type Marshaled struct {
	Ids []string `json:"ids"`
}

func main() {

	var countries []Country
	dataFromFile, err := ioutil.ReadFile("countries.json")

	if err != nil {
		log.Fatal("error while tryna open file")
	}

	err = json.Unmarshal(dataFromFile, &countries)

	if err != nil {
		log.Fatal("unmarshalling error")
	}

	var cityIds []string

	for _, v := range countries {
		str := fmt.Sprintf("%s", v.Id)

		if str == "" {
			continue
		}

		cityIds = append(cityIds, str)
	}

	var toMarshal Marshaled
	toMarshal.Ids = cityIds

	marshaled, err := json.Marshal(toMarshal)

	f, err := os.Create("optimizedCountriesCityId.json")
	defer f.Close()

	_, err = f.Write(marshaled)
	if err != nil {
		log.Fatal("failed to write to file")
	}

}
