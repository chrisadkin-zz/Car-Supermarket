package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"goji.io"
	"goji.io/pat"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func errorWithJSON(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	fmt.Fprintf(w, "{message: %q}", message)
}

func responseWithJSON(w http.ResponseWriter, json []byte, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(json)
}

type vehicle struct {
	Manurfacturer string `json:"manufacturer"`
	Model         string `json:"model"`
	VIN           string `json:"vin"`
	RegNo         string `json:"regno"`
}

func main() {
	session, err := mgo.Dial("mongo")

	if err != nil {
		panic(err)
	}

	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	ensureIndex(session)

	mux := goji.NewMux()
	mux.HandleFunc(pat.Get("/cars"), allCars(session))
	mux.HandleFunc(pat.Post("/cars"), addCar(session))
	mux.HandleFunc(pat.Get("/cars/:vin"), carByVIN(session))
	mux.HandleFunc(pat.Delete("/cars/:vin"), deleteCar(session))
	http.ListenAndServe(":8080", mux)
}

func ensureIndex(s *mgo.Session) {
	session := s.Copy()
	defer session.Close()

	c := session.DB("carsupermarket").C("cars")

	index := mgo.Index{
		Key:        []string{"vin"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	err := c.EnsureIndex(index)
	if err != nil {
		panic(err)
	}
}

func allCars(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		c := session.DB("carsupermarket").C("cars")

		var cars []vehicle
		err := c.Find(bson.M{}).All(&cars)
		if err != nil {
			errorWithJSON(w, "Database error", http.StatusInternalServerError)
			log.Println("Failed get all cars: ", err)
			return
		}

		respBody, err := json.MarshalIndent(cars, "", "  ")
		if err != nil {
			log.Fatal(err)
		}

		responseWithJSON(w, respBody, http.StatusOK)
	}
}

func addCar(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		var car vehicle
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&car)
		if err != nil {
			errorWithJSON(w, "Incorrect body", http.StatusBadRequest)
			return
		}

		c := session.DB("carsupermarket").C("cars")

		err = c.Insert(car)
		if err != nil {
			if mgo.IsDup(err) {
				errorWithJSON(w, "A car with this VIN already exists", http.StatusBadRequest)
				return
			}

			errorWithJSON(w, "Database error", http.StatusInternalServerError)
			log.Println("Failed insert car: ", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Location", r.URL.Path+"/"+car.VIN)
		w.WriteHeader(http.StatusCreated)
	}
}

func carByVIN(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		vin := pat.Param(r, "vin")

		c := session.DB("carsupermarket").C("cars")

		var car vehicle
		err := c.Find(bson.M{"vin": vin}).One(&car)
		if err != nil {
			errorWithJSON(w, "Database error", http.StatusInternalServerError)
			log.Println("Failed find car: ", err)
			return
		}

		if car.VIN == "" {
			errorWithJSON(w, "Car not found", http.StatusNotFound)
			return
		}

		respBody, err := json.MarshalIndent(car, "", "  ")
		if err != nil {
			log.Fatal(err)
		}

		responseWithJSON(w, respBody, http.StatusOK)
	}
}

func deleteCar(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		vin := pat.Param(r, "vin")

		c := session.DB("carsupermarket").C("cars")

		err := c.Remove(bson.M{"vin": vin})
		if err != nil {
			switch err {
			default:
				errorWithJSON(w, "Database error", http.StatusInternalServerError)
				log.Println("Failed delete car: ", err)
				return
			case mgo.ErrNotFound:
				errorWithJSON(w, "Car not found", http.StatusNotFound)
				return
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
