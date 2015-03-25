package main

import (
	"encoding/json"
	"fmt"
	"github.com/scorredoira/email"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"time"
)

type Config struct {
	Endpoint      string
	Interval      string
	Target        string
	Threshold     float64
	Frequency     string
	Rule          string
	EmailServer   string
	EmailTo       string
	EmailFrom     string
	EmailUser     string
	EmailPassword string
	EmailPort     string
	EmailSubject  string
}

type Data struct {
	Target     string
	DataPoints [][2]float64
}

type Alarm struct {
	Target    string
	Rule      string
	Threshold float64
}

func main() {
	out, err := os.Create("graphmon.log")
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()
	log.SetOutput(out)

	config := readConfig()
	auth := smtp.PlainAuth("", config.EmailUser, config.EmailPassword, config.EmailServer)
	d, err := time.ParseDuration(config.Frequency)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if r := recover(); r != nil {
			sendEmail(config.EmailServer+":"+config.EmailPort, auth, "graphite-monitor encountered an error: "+err.Error(), config.EmailTo, config.EmailFrom)
		}
	}()
	for {

		data := getData(config)
		alarms := monitorData(data, config.Rule, config.Threshold)
		for i := range alarms {
			fmt.Printf("Target: %s has not met the threshold %f\n", alarms[i].Target, alarms[i].Threshold)
			name := saveGraph(alarms[i], config)
			sendEmailwithAttachment(config.EmailServer+":"+config.EmailPort, auth, config.EmailSubject+" "+alarms[i].Target, config.EmailTo, config.EmailFrom, name)
			os.Remove(name)
		}
		time.Sleep(d)
	}
}

func sendEmailwithAttachment(addr string, auth smtp.Auth, subject string, to string, from string, filename string) {
	m := email.NewMessage(subject, "")
	m.To = []string{to}
	m.From = from
	err := m.Attach(filename)
	if err != nil {
		log.Panic(err)
	}
	err = email.Send(addr, auth, m)
	if err != nil {
		log.Panic(err)
	}
}

func sendEmail(addr string, auth smtp.Auth, subject string, to string, from string) {
	m := email.NewMessage(subject, "")
	m.To = []string{to}
	m.From = from
	err := email.Send(addr, auth, m)
	if err != nil {
		log.Panic(err)
	}
}

func saveGraph(alarm Alarm, config Config) string {
	var graphurl = config.Endpoint + "/render?" + "target=" + alarm.Target + "&from=" + config.Interval
	out, err := os.Create(time.Now().Format("01-02-2015T15.04.05") + ".png")
	if err != nil {
		log.Panic(err)
	}
	defer out.Close()
	resp, err := http.Get(graphurl)
	if err != nil {
		log.Panic(err)
	}
	defer resp.Body.Close()
	io.Copy(out, resp.Body)
	return out.Name()
}

func monitorData(d []Data, rule string, thres float64) []Alarm {
	alarms := make([]Alarm, 0)
	for i := range d {
		data := d[i]
		alarm := Alarm{}
		switch rule {
		case "==":
			for j := range data.DataPoints {
				if data.DataPoints[j][0] == thres {
					alarm.Threshold = thres
					alarm.Target = data.Target
					alarm.Rule = rule
					alarms = append(alarms, alarm)
					break
				}
			}
		case "!=":
			for j := range data.DataPoints {
				if data.DataPoints[j][0] != thres {
					alarm.Threshold = thres
					alarm.Target = data.Target
					alarm.Rule = rule
					alarms = append(alarms, alarm)
					break
				}
			}
		case "<":
			for j := range data.DataPoints {
				if data.DataPoints[j][0] < thres {
					alarm.Threshold = thres
					alarm.Target = data.Target
					alarm.Rule = rule
					alarms = append(alarms, alarm)
					break
				}
			}
		case "<=":
			for j := range data.DataPoints {
				if data.DataPoints[j][0] <= thres {
					alarm.Threshold = thres
					alarm.Target = data.Target
					alarm.Rule = rule
					alarms = append(alarms, alarm)
					break
				}
			}
		case ">":
			for j := range data.DataPoints {
				if data.DataPoints[j][0] > thres {
					alarm.Threshold = thres
					alarm.Target = data.Target
					alarm.Rule = rule
					alarms = append(alarms, alarm)
					break
				}
			}
		case ">=":
			for j := range data.DataPoints {
				if data.DataPoints[j][0] >= thres {
					alarm.Threshold = thres
					alarm.Target = data.Target
					alarm.Rule = rule
					alarms = append(alarms, alarm)
					break
				}
			}
		default:
			log.Fatal("the rule cannot be parsed!")
		}
	}

	return alarms
}

func readConfig() Config {
	file, _ := os.Open("conf.json")
	decoder := json.NewDecoder(file)
	configuration := Config{}
	err := decoder.Decode(&configuration)
	if err != nil {
		log.Fatal(err)
	}
	return configuration
}

func getData(config Config) []Data {
	ep, _ := url.Parse(config.Endpoint)
	values := url.Values{}
	values.Set("target", config.Target)
	values.Add("from", config.Interval)
	actualurl := ep.String() + "/render" + "?" + values.Encode() + "&format=json"

	resp, err := http.Get(actualurl)
	defer resp.Body.Close()
	if err != nil {
		log.Panic(err)
	}
	dec := json.NewDecoder(resp.Body)
	var m []Data
	for {
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			log.Panic(err)
		}
	}
	return m
}
