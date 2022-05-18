// Phoronix Test Results Parser

package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	PNAME            = "ptrp"
	outputDirDefault = "./"
	noiseMaxDefault  = 3.5 // in percent
	suiteNameDefault = "suite"

	gp = `
set output graph_image_out
set datafile separator "|"
set datafile missing "-"

# Set the title into a fn()?
if (test_proportion eq "LIB") {
  proportion_ch="<"
  proportion_text="Lower is better"
} else {
  proportion_ch=">"
  proportion_text="Higher is better"
}
legend="^{{/:Bold " . proportion_ch . " }{/:Italics " . proportion_text . " [" . test_units . "]}}"
set title "{/=12 " . test_name . "}\n_{/=10 " . test_description . "}\n\n" . legend

set grid
set colors podo		# friendly to color blind individual
set key off		# turn off all titles (or unset key)

set yrange [0:*]	# start at zero, find max from the data
set style fill solid	# solid color boxes
myBoxWidth=0.8
set offsets 0,0,0.7-myBoxWidth/2.,0.7

plot \
  $graph_data_in using 2:0:(0):2:($0-myBoxWidth/2.):($0+myBoxWidth/2.):($0+1):ytic(1) with boxxyerror lc var notitle
`
)

// Global variables.
var (
	pOutputDir = outputDirDefault
	pNoiseMax  = noiseMaxDefault
	pSuiteName = suiteNameDefault
	pFormat    = "text"
	pGnuplot   = flag.Bool("g", false, "output gnuplot file(s) processing")
	pScore     = flag.Bool("s", false, "output only scoring statistics based on better performance")
)

type custom struct {
	label string
	score int
}

// Results is a struct which contains the complete slice of all Results.
type PhoronixTestSuite struct {
	XMLName   xml.Name  `xml:"PhoronixTestSuite"`
	Generated Generated `xml:"Generated"`
	System    System    `xml:"System"`
	Results   []Result  `xml:"Result"`
	c         custom
}

type Generated struct {
	XMLName      xml.Name `xml:"Generated"`
	Title        string   `xml:"Title"`
	LastModified string   `xml:"LastModified"`
	TestClient   string   `xml:"TestClient"`
	Description  string   `xml:"Description"`
}

type System struct {
	XMLName           xml.Name `xml:"System"`
	Identifier        string   `xml:"Identifier"`
	Hardware          string   `xml:"Hardware"`
	Software          string   `xml:"Software"`
	User              string   `xml:"User"`
	TimeStamp         string   `xml:"TimeStamp"`
	TestClientVersion string   `xml:"TestClientVersion"`
	Notes             string   `xml:"Notes"`
	JSON              string   `xml:"JSON"`
}

type Result struct {
	XMLName       xml.Name `xml:"Result"`
	Identifier    string   `xml:"Identifier"`
	Title         string   `xml:"Title"`
	AppVersion    string   `xml:"AppVersion"`
	Arguments     string   `xml:"Arguments"`
	Description   string   `xml:"Description"`
	Scale         string   `xml:"Scale"`
	Proportion    string   `xml:"Proportion"`
	DisplayFormat string   `xml:"DisplayFormat"`
	Data          Data     `xml:"Data"`
}

type Data struct {
	XMLName xml.Name `xml:"Data"`
	Entry   Entry    `xml:"Entry"`
}

type Entry struct {
	XMLName    xml.Name `xml:"Entry"`
	Identifier string   `xml:"Identifier"`
	Value      string   `xml:"Value"`
	RawString  string   `xml:"RawString"`
	JSON       string   `xml:"JSON"`
}

func splitWithEsc(s string, separator, escape byte) []string {
	var (
		buf []byte
		ret []string
	)
	for i := 0; i < len(s); i++ {
		if s[i] == separator {
			ret = append(ret, string(buf))
			buf = buf[:0]
		} else if s[i] == escape && i+1 < len(s) {
			i++
			buf = append(buf, s[i])
		} else {
			buf = append(buf, s[i])
		}
	}
	ret = append(ret, string(buf))

	return ret
}

func parseCmdOpts() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <composite.xml file(s)>\n", PNAME)
		fmt.Fprintf(os.Stderr, "Example: %s -o out a/composite.xml b/composite.xml\n\n", PNAME)

		flag.PrintDefaults()
	}

	flag.StringVar(&pFormat, "f", pFormat, "format: csv|gp|text")
	flag.Float64Var(&pNoiseMax, "noise-max", pNoiseMax, "allow for some noise")
	flag.StringVar(&pOutputDir, "o", pOutputDir, "output directory to write gnuplot files to")
	flag.StringVar(&pSuiteName, "suite-name", pSuiteName, "suite name to prefix the statistics scores with")
	flag.Parse()
}

func parseResultsXml(fileIn string) (PhoronixTestSuite, error) {
	var pts PhoronixTestSuite

	xmlFile, err := os.Open(fileIn)
	if err != nil {
		return pts, fmt.Errorf("failed to open XML file %s: %v", fileIn, err)
	}

	defer xmlFile.Close()

	byteSlice, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		return pts, fmt.Errorf("failed to read XML file %s: %v", fileIn, err)
	}

	err = xml.Unmarshal(byteSlice, &pts)
	if err != nil {
		return pts, fmt.Errorf("failed to unmarshal XML file %s: %v", fileIn, err)
	}

	return pts, nil
}

func sanitizeTestIdentifier(id string) string {
	i := strings.LastIndex(id, "/")
	if i == -1 {
		i = 0
	}
	return id[i+1:]
}

func resultsToText(pts []PhoronixTestSuite) error {
	if len(pts) == 0 {
		return fmt.Errorf("no PTS results")
	}

	results := len(pts[0].Results)
	fmtWide := fmt.Sprintf("%%0%dd", len(fmt.Sprintf("%d", results)))

	for n, r := range pts[0].Results {
		fmt.Printf(fmtWide+"-%s.gp\n", n, sanitizeTestIdentifier(r.Identifier))
		fmt.Printf("========================\n")
		fmt.Printf("Identifier: %s\n", r.Identifier)
		fmt.Printf("Title: %s\n", r.Title)
		fmt.Printf("Description: %s\n", r.Description)
		fmt.Printf("Scale: %s\n", r.Scale)
		fmt.Printf("Proportion: %s\n", r.Proportion)
		fmt.Printf("AppVersion: %s\n", r.AppVersion)

		fmt.Printf("  %s\n", pts[0].Results[n].Data.Entry.Value)
		for i := 1; i < len(pts); i++ {
			value := ""
			found := false
			o := 0
			for ; o < len(pts[i].Results); o++ {
				if r.Identifier != pts[i].Results[o].Identifier ||
					r.Arguments != pts[i].Results[o].Arguments {
					continue
				}
				value = pts[i].Results[o].Data.Entry.Value
				found = true
			}
			if !found || len(value) == 0 {
				value = "-"
			}
			fmt.Printf("  %s\n", value)
		}
		fmt.Printf("\n")
	}

	return nil
}

func resultsToCSV(pts []PhoronixTestSuite) error {
	if len(pts) == 0 {
		return fmt.Errorf("no PTS results")
	}

	for _, s := range pts {
		fmt.Printf("# %s\n", s.c.label)
		fmt.Printf("# ========================\n")
		for _, r := range s.Results {
			fmt.Printf("%s: %s,%s,%s\n", r.Title, r.Description, r.Proportion, r.Data.Entry.Value)
		}
		fmt.Println()
	}

	return nil
}

func resultsToGnuplot(pts []PhoronixTestSuite) error {
	var sb strings.Builder

	if len(pts) == 0 {
		return fmt.Errorf("no PTS results")
	}

	results := len(pts[0].Results)
	fmtWide := fmt.Sprintf("%%0%dd", len(fmt.Sprintf("%d", results)))

	for n, r := range pts[0].Results {
		if len(pts[0].Results[n].Data.Entry.Value) == 0 {
			continue
		}

		sb.Reset()
		basename := fmt.Sprintf(fmtWide+"-%s", n, sanitizeTestIdentifier(r.Identifier))
		outputFile := basename + ".gp"
		f, err := os.Create(pOutputDir + "/" + outputFile)
		if err != nil {
			return err
		}
		defer f.Close()

		sb.WriteString("$graph_data_in <<EOD\n")
		sb.WriteString(fmt.Sprintf("%s\\n{/:Italics{%s}|%s\n", pts[0].c.label, pts[0].Results[n].Data.Entry.Value, pts[0].Results[n].Data.Entry.Value))
		for i := 1; i < len(pts); i++ {
			value := ""
			found := false
			o := 0
			for ; o < len(pts[i].Results); o++ {
				if r.Identifier != pts[i].Results[o].Identifier ||
					r.Arguments != pts[i].Results[o].Arguments {
					continue
				}
				value = pts[i].Results[o].Data.Entry.Value
				found = true
			}
			if !found || len(value) == 0 {
				value = "-"
			}
			sb.WriteString(fmt.Sprintf("%s\\n{/:Italics{%s}|%s\n", pts[i].c.label, value, value))
		}
		sb.WriteString("EOD\n\n")

		sb.WriteString("# Test-specific variables\n")
		sb.WriteString(fmt.Sprintf("graph_image_out=\"%s.png\"\n", basename))
		sb.WriteString(fmt.Sprintf("test_name=\"%s\"\n", r.Title))
		sb.WriteString(fmt.Sprintf("test_description=\"%s\"\n", r.Description))
		sb.WriteString(fmt.Sprintf("test_units=\"%s\"\n", r.Scale))
		sb.WriteString(fmt.Sprintf("test_proportion=\"%s\"\n", r.Proportion))
		sb.WriteString(fmt.Sprintf("test_version=\"%s\"\n", r.AppVersion))

		sb.WriteString(fmt.Sprintf("set terminal png enhanced size 800,%d font ',10'\n", 120+40*len(pts)))

		sb.WriteString(gp)

		_, err = f.WriteString(sb.String())
		if err != nil {
			return err
		}

		if *pGnuplot {
			_, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("cd %s && gnuplot %s", pOutputDir, outputFile)).Output()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err)
			}
		}
	}

	return nil
}

func mkdir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

func score(pts []PhoronixTestSuite) error {
	var values []float64

	if len(pts) == 0 {
		return fmt.Errorf("no PTS results")
	}

	parseFloat := func(s string) float64 {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			f = 0
		}
		return f
	}

	suites := len(pts)

loop:
	for n, r := range pts[0].Results {
		values = make([]float64, suites)
		values[0] = parseFloat(pts[0].Results[n].Data.Entry.Value)
		if len(pts[0].Results[n].Data.Entry.Value) == 0 || values[0] == 0 {
			continue
		}

		for i := 1; i < suites; i++ {
			found := false
			o := 0
			for ; o < len(pts[i].Results); o++ {
				if r.Identifier != pts[i].Results[o].Identifier ||
					r.Arguments != pts[i].Results[o].Arguments {
					continue
				}
				values[i] = parseFloat(pts[i].Results[o].Data.Entry.Value)
				found = true
			}
			if !found || values[i] == 0 {
				continue
			}
		}

		// Set the scores only if we have non-zero values for all results.
		for i := 0; i < suites; i++ {
			if values[i] == 0 {
				continue loop
			}
		}

		// Set the scores.
		for i := 0; i < suites; i++ {
			for j := 0; j < suites; j++ {
				if i == j {
					continue
				}
				// Allow for some noise.
				noise := values[i]*(1+pNoiseMax/100) - values[i]
				if pts[0].Results[n].Proportion == "LIB" {
					// Lower is better.
					if values[i]-noise <= values[j] {
						pts[i].c.score++
					}
				} else {
					// Higher is better.
					if values[i]+noise >= values[j] {
						pts[i].c.score++
					}
				}
			}
		}
	}

	// Gnuplot friendly output.
	fmt.Print("# Suite")
	for i := 0; i < suites; i++ {
		fmt.Printf("\t%v", pts[i].c.label)
	}
	fmt.Println()

	fmt.Printf("%v", pSuiteName)
	for i := 0; i < suites; i++ {
		fmt.Printf("\t%v", pts[i].c.score)
	}
	fmt.Println()

	return nil
}

func parsePTSResults() error {
	var pts []PhoronixTestSuite

	for i, arg := range flag.Args() {
		a := splitWithEsc(arg, '|', '\\')
		r, err := parseResultsXml(a[0])
		if err != nil {
			return err
		}

		pts = append(pts, r)
		if len(a) > 1 {
			pts[i].c.label = a[1]
		}
	}

	if pOutputDir != outputDirDefault {
		err := mkdir(pOutputDir)
		if err != nil {
			return err
		}
	}

	if *pScore {
		return score(pts)
	}

	switch pFormat {
	case "csv":
		return resultsToCSV(pts)
	case "gp":
		return resultsToGnuplot(pts)
	case "text":
		return resultsToText(pts)
	default:
		return resultsToText(pts)
	}
}

func main() {
	parseCmdOpts()

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	err := parsePTSResults()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse test results: %v\n", err)
	}
}
