package gx


import (
	"fmt"
	"time"
)

// ThaiMonthNames maps month numbers to their Thai names.
var ThaiMonthNames = []string{
	"มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน", "พฤษภาคม", "มิถุนายน",
	"กรกฎาคม", "สิงหาคม", "กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม",
}

// FormatThaiDate formats the given time as "2 กรกฎาคม 2568" (Thai date format).
func FormatThaiDate(t time.Time) string {
	// Convert the year to Thai calendar (add 543 years)
	thaiYear := t.Year() + 543

	// Get the day and month
	day := t.Day()
	month := ThaiMonthNames[t.Month()-1]

	// Return the formatted date
	return fmt.Sprintf("%d %s %d", day, month, thaiYear)
}

// ParseAndFormatThaiDate parses a given RFC3339 date string and formats it in Thai date format.
func ParseAndFormatThaiDate(dateStr string) (string, error) {
	// Parse the date string in RFC3339 format (Bangkok time)
	parsedDate, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return "", err
	}

	// Load Bangkok location and convert to Bangkok local time
	bangkokLocation, _ := time.LoadLocation("Asia/Bangkok")
	parsedDate = parsedDate.In(bangkokLocation)

	// Format the parsed date in Thai format
	return FormatThaiDate(parsedDate), nil
}
