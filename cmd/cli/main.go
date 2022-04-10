package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

type ScrambleStreams struct {
	Streams map[string]Stream
}

type TagOptions struct {
	Name       string
	MultiValue string
	SubValue   string
	Pattern    string
}

type Stream struct {
	Tables []string
	Tags   []TagOptions
}

type T24FieldMS struct {
	Text string `xml:",chardata"`
	M    string `xml:"m,attr"`
	S    string `xml:"s,attr"`
}

type T24FieldM struct {
	Text string `xml:",chardata"`
	M    string `xml:"m,attr"`
}

type T24Field struct {
	Text string `xml:",chardata"`
}

type TagKey struct {
	Name       string
	MultiValue string
	SubValue   string
}

// IfThenElse evaluates a condition, if true returns the first parameter otherwise the second
func IfThenElse(condition bool, a interface{}, b interface{}) interface{} {
	if condition {
		return a
	}
	return b
}

func main() {
	args := os.Args
	if len(args) < 2 {
		log.Println("usage exe <streams.json> <connection string>")
		return
	}
	fmt.Println(args)

	fileStreams, err := ioutil.ReadFile(args[1])
	if err != nil {
		log.Fatalln(err)
		return
	}

	// Read & show streams
	var streams map[string]Stream
	json.Unmarshal(fileStreams, &streams)
	for streamName, stream := range streams {
		log.Printf("Stream %s\n", streamName)
		for _, tableName := range stream.Tables {
			log.Printf("Table: %s\n", tableName)
		}
	}

	// Build FT tags map
	a := make(map[TagKey]string, len(streams["ft"].Tags))
	for _, v := range streams["ft"].Tags {
		key := TagKey{
			Name:       v.Name,
			MultiValue: fmt.Sprint(IfThenElse(v.MultiValue == "", "0", v.MultiValue)),
			SubValue:   fmt.Sprint(IfThenElse(v.SubValue == "", "0", v.SubValue)),
		}
		a[key] = v.Pattern
	}

	// Open FT.xml & print structure
	filexml, err := os.Open("..\\..\\ft.xml")
	if err != nil {
		log.Fatal(err)
	}
	defer filexml.Close()

	var buf bytes.Buffer

	decoder := xml.NewDecoder(filexml)
	encoder := xml.NewEncoder(&buf)

	recId := "temp id"

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("error getting token: %v\n", err)
			break
		}

		switch v := token.(type) {
		case xml.StartElement:
			elmt := xml.StartElement(v)

			// Process only <c...> elements
			if elmt.Name.Local == "row" {
				break
			}
			var elementM string = "1"
			var elementS string = "1"
			for _, v := range elmt.Attr {
				switch v.Name.Local {
				case "m":
					elementM = v.Value
				case "s":
					elementS = v.Value
				}
			}

			// Get pattern for the element (0, 0)
			var pattern string
			var tagKey TagKey
			tagKey = TagKey{Name: elmt.Name.Local, MultiValue: "0", SubValue: "0"}
			pattern, ok := a[tagKey]
			if !ok {
				// Get pattern for the element (m, 0)
				tagKey = TagKey{Name: elmt.Name.Local, MultiValue: elementM, SubValue: "0"}
				pattern, ok = a[tagKey]
				if !ok {
					// Get pattern for the element (0, 0)
					tagKey = TagKey{Name: elmt.Name.Local, MultiValue: elementM, SubValue: elementS}
					pattern, ok = a[tagKey]
					if !ok {
						break
					}
				}
			}

			if elementM == "1" && elementS == "1" {
				var field T24Field
				if err = decoder.DecodeElement(&field, &v); err != nil {
					log.Fatal(err)
				}
				field.Text = fmt.Sprintf(pattern, recId, elementM, elementS)
				if err = encoder.EncodeElement(field, v); err != nil {
					log.Fatal(err)
				}
			} else if elementS == "1" {
				var field T24FieldM
				if err = decoder.DecodeElement(&field, &v); err != nil {
					log.Fatal(err)
				}
				field.Text = fmt.Sprintf(pattern, recId, elementM, elementS)
				if err = encoder.EncodeElement(field, v); err != nil {
					log.Fatal(err)
				}
			} else {
				var field T24FieldMS
				if err = decoder.DecodeElement(&field, &v); err != nil {
					log.Fatal(err)
				}
				field.Text = fmt.Sprintf(pattern, recId, elementM, elementS)
				if err = encoder.EncodeElement(field, v); err != nil {
					log.Fatal(err)
				}
			}
			continue
		}
		if err := encoder.EncodeToken(xml.CopyToken(token)); err != nil {
			log.Fatal(err)
		}
	}
	encoder.Flush()
	fmt.Println(buf.String())
}
