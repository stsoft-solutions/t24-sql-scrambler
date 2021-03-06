package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

// IfThenElse evaluates a condition, if true returns the first parameter otherwise the second
func IfThenElse(condition bool, a interface{}, b interface{}) interface{} {
	if condition {
		return a
	}
	return b
}

type Database struct {
	SqlDb *sql.DB
}

func readTransactions(db *sql.DB) error {

	startedAt := time.Now()

	rows, err := db.Query("SELECT RECNO, XMLRECORD FROM [dbo].[FundsTransfers]")
	if err != nil {
		fmt.Println("Query error: ", err.Error())
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	if cols == nil {
		return nil
	}

	vals := make([]interface{}, len(cols))
	for i := 0; i < len(cols); i++ {
		vals[i] = new(interface{})
		if i != 0 {
			fmt.Print("\t")
		}
		fmt.Print(cols[i])
	}
	fmt.Println()

	for rows.Next() {
		var recordId string
		var xmlBody string
		err = rows.Scan(&recordId, &xmlBody)
		if err != nil {
			fmt.Println(err)
			continue
		}

		//fmt.Println(tranNumber)

	}

	duration := time.Since(startedAt)

	fmt.Printf("Reading is finished %s\n", duration)

	return nil
}

func main() {

	connectionString := "sqlserver://sa:Qwerty123!@localhost?database=TranDb&connection+timeout=30"

	db, err := sql.Open("mssql", connectionString)
	if err != nil {
		fmt.Println("Cannot connect: ", err.Error())
		return
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		fmt.Println("Cannot connect: ", err.Error())
		return
	}

	readTransactions(db)

	//return

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
	filexml, err := os.Open("ft.xml")
	if err != nil {
		log.Fatal(err)
	}
	defer filexml.Close()

	var buf bytes.Buffer

	decoder := xml.NewDecoder(filexml)
	encoder := xml.NewEncoder(&buf)

	recId := "temp id"
	var isChanged bool
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
				isChanged = true
			} else if elementS == "1" {
				var field T24FieldM
				if err = decoder.DecodeElement(&field, &v); err != nil {
					log.Fatal(err)
				}
				field.Text = fmt.Sprintf(pattern, recId, elementM, elementS)
				if err = encoder.EncodeElement(field, v); err != nil {
					log.Fatal(err)
				}
				isChanged = true
			} else {
				var field T24FieldMS
				if err = decoder.DecodeElement(&field, &v); err != nil {
					log.Fatal(err)
				}
				field.Text = fmt.Sprintf(pattern, recId, elementM, elementS)
				if err = encoder.EncodeElement(field, v); err != nil {
					log.Fatal(err)
				}
				isChanged = true
			}
			continue
		}
		if err := encoder.EncodeToken(xml.CopyToken(token)); err != nil {
			log.Fatal(err)
		}
	}
	encoder.Flush()
	fmt.Println(buf.String())
	fmt.Println(isChanged)
}
