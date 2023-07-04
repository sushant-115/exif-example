package main

import (
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	exif "github.com/dsoprea/go-exif"
	gocsv "github.com/gocarina/gocsv"
)

type GPSInfo struct {
	FilePath  string  `csv:"image file path"`
	Longitude float64 `csv:"GPS position latitude"`
	Latitude  float64 `csv:"GPS position longitude"`
}

// recover function to handle panic
func handlePanic(filepath string) {
	// detect if panic occurs or not
	a := recover()
	if a != nil {
		log.Println("RECOVER", a, filepath)
	}
}

func getExifGPSInfo(filepath string) *GPSInfo {
	defer handlePanic(filepath)
	var result *GPSInfo
	rawExif, err := exif.SearchFileAndExtractExif(filepath)
	if err != nil {
		log.Println("error occured during search: ", err)
		return nil
	}
	im := exif.NewIfdMappingWithStandard()
	ti := exif.NewTagIndex()
	_, index, err := exif.Collect(im, ti, rawExif)
	if err != nil {
		log.Println("error occured during collect for file: ", filepath, "error: ", err)
		return result
	}
	ifds, ok := index.Lookup[exif.IfdPathStandardGps]
	if !ok {
		return result
	}
	for _, ifd := range ifds {
		if ifd.Name == exif.IfdGps {
			gpsInfo, _ := ifd.GpsInfo()
			result = &GPSInfo{}
			result.FilePath = filepath
			result.Longitude = gpsInfo.Longitude.Decimal()
			result.Latitude = gpsInfo.Latitude.Decimal()
		}
	}
	return result
}

func filePathWalkDir(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func writeContentToFile(results []*GPSInfo, outputFile string, outputFormat string) error {
	switch outputFormat {
	case "csv":
		csvContent, err := gocsv.MarshalString(results)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(outputFile, []byte(csvContent), 0664)
		return err
	case "html":
		tmpl, err := template.ParseFiles("./html-templates/index.html")
		if err != nil {
			log.Fatalln(err)
		}
		f, err := os.OpenFile(outputFile, os.O_RDWR, os.ModeAppend)
		if err != nil {
			log.Fatalln(err)
		}
		err = tmpl.Execute(f, results)
		if err != nil {
			log.Fatalln(err)
		}
	}
	return nil
}

func main() {
	root := flag.String("path", "/Users/sushant/Downloads/images", "Path to read the exif data")
	outputFormat := flag.String("output-format", "csv", "Format of output. Either csv or html")
	outputFile := flag.String("output-file", "/tmp/test.csv", "Write the result to this file")
	// recursive := flag.Bool("recursive", false, "Read the directory recursively")
	flag.Parse()
	log.SetOutput(os.Stderr)

	// List files in the directory
	files, err := filePathWalkDir(*root)
	if err != nil {
		log.Fatal("error while listing the directory:", err)
	}
	var results []*GPSInfo
	for _, file := range files {
		result := getExifGPSInfo(file)
		if result == nil {
			continue
		}
		results = append(results, result)
	}
	err = writeContentToFile(results, *outputFile, *outputFormat)
	if err != nil {
		log.Println("error occured while writing content", err, *outputFormat)
	}
}
