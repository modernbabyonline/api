package main

import (
	"fmt"

	"github.com/tidwall/gjson"
)

func parseCalendlyJSON(info string) string {
	fmt.Println("parsing info")
	value := gjson.Get(info, "body")
	fmt.Println(value)
	return value.String()

}
