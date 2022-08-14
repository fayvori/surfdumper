package main

import (
	"SurfHotelsDumper/constants"
	"SurfHotelsDumper/hasher"
	"SurfHotelsDumper/models"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	client = resty.New()
	Ctx    = context.TODO()
)

func AddPhotosToHotelDbResponse(hotels *models.HotelResponse) {
	var hotelsIds []string

	if len(hotels.Result) > 200 {
		for i := 0; i < 200; i++ {
			hotelsIds = append(hotelsIds, strconv.Itoa(hotels.Result[i].Id))
		}
	} else {
		for i := 0; i < len(hotels.Result); i++ {
			hotelsIds = append(hotelsIds, strconv.Itoa(hotels.Result[i].Id))
		}
	}

	resp, err := client.R().
		SetQueryParams(map[string]string{
			"id": strings.Join(hotelsIds, ","),
		}).
		Get("https://yasen.hotellook.com/photos/hotel_photos")

	if err != nil {
		fmt.Println(err.Error())
	}

	var photos map[string][]int
	err = json.Unmarshal(resp.Body(), &photos)

	for i := 0; i < len(hotels.Result); i++ {
		id := strconv.Itoa(hotels.Result[i].Id)
		id1, _ := strconv.Atoi(id)

		if hotels.Result[i].Id == id1 {
			hotels.Result[i].PhotoHotel = photos[id]
		}
	}
}

const (
	currency    string = "USD"
	language    string = "ru"
	datePattern string = "2006-01-02"
)

var (
	today     time.Time = time.Now().UTC()
	startDate time.Time = today.AddDate(0, 0, 1)
	endDate   time.Time = today.AddDate(0, 0, 11)
)

type MarshaledCityIds struct {
	CityIds []string `json:"ids"`
}

func main() {
	const uri = "mongodb://localhost:27017"
	connect, err := mongo.Connect(Ctx, options.Client().ApplyURI(uri))

	if err != nil {
		log.Printf(err.Error())
	}

	defer func() {
		if err := connect.Disconnect(Ctx); err != nil {
			panic(err)
		}
	}()

	if err := connect.Ping(Ctx, readpref.Primary()); err != nil {
		log.Fatalf(err.Error())
	}

	coll := connect.Database("surf-hotelDumper").Collection("hotels")

	var cityIds MarshaledCityIds

	f, err := ioutil.ReadFile("optimizedCountriesCityId.json")

	if err != nil {
		log.Fatal("Failed to open optimizedCountriesCityId.json file")
	}

	err = json.Unmarshal(f, &cityIds)

	//: TODO the max is 200-1 with 1,5 hours break
	for key, cityId := range cityIds.CityIds {
		hotelsHash := hasher.Md5HotelHasher(fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s:%s:%s:%s",
			constants.TOKEN,
			constants.MARKER,
			"1",
			startDate.Format(datePattern),
			endDate.Format(datePattern),
			cityId,
			currency,
			constants.CUSTOMER_IP,
			language,
			"1",
		))

		// cityIdParsed, _ := strconv.Atoi(cityId)

		respSearchId, err := client.R().
			EnableTrace().
			Get(fmt.Sprintf("%s/start.json?cityId=%s&checkIn=%s&checkOut=%s&adultsCount=%s&customerIP=%s&lang=%s&currency=%s&waitForResult=%s&marker=%s&signature=%s",
				constants.HOTELLOOK_ADDR,
				cityId,
				startDate.Format(datePattern),
				endDate.Format(datePattern),
				"1",
				constants.CUSTOMER_IP,
				language,
				currency,
				"1",
				constants.MARKER,
				hotelsHash,
			))

		fmt.Println(string(respSearchId.Body()))

		if err != nil {
			log.Printf(err.Error())
		}

		var hotels models.HotelResponse
		err = json.Unmarshal(respSearchId.Body(), &hotels)
		if err != nil {
			log.Printf(err.Error())
		}

		fmt.Println(len(hotels.Result))
		time.Sleep(15 * time.Second)

		if len(hotels.Result) > 0 {
			sort.SliceStable(hotels.Result, func (i, j int) bool {
				return hotels.Result[i].Price < hotels.Result[j].Price
			})
		
			sort.SliceStable(hotels.Result, func (i, j int) bool {
				return hotels.Result[i].Stars > hotels.Result[j].Stars
			})

			var lengthToInsert int

			if len(hotels.Result) > 300 {
				lengthToInsert = 300
			} else {
				lengthToInsert = len(hotels.Result)
			}

			AddPhotosToHotelDbResponse(&hotels)

			fmt.Println("key: ", key)
			fmt.Println("length to insert: ", lengthToInsert)

			for _, v := range hotels.Result[:lengthToInsert] {
				// set params for searching and optimize sizes of rooms to 1 room
				v.CityId = cityId
				v.Lang   = language
				v.Rooms  = []models.HotelRoom{v.Rooms[0]}

				result, err := coll.InsertOne(constants.Ctx, v)
				if err != nil {
					log.Printf(err.Error())
				}

				fmt.Printf("Inserted %d\n", result.InsertedID)
			}
		}
	}
}