package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

const version = "1.0"

var dpNumPattern = regexp.MustCompile(`(\d+)dp`)
var colorPattern = regexp.MustCompile(`^#([0-9a-fA-F]{3,8})$`)

type vector struct {
	XMLName        xml.Name
	Width          string       `xml:"width,attr"`
	Height         string       `xml:"height,attr"`
	ViewportWidth  float64      `xml:"viewportWidth,attr"`
	ViewportHeight float64      `xml:"viewportHeight,attr"`
	Paths          []vectorPath `xml:"path"`
}

type vectorPath struct {
	FillColor   string  `xml:"fillColor,attr"`
	StrokeColor string  `xml:"strokeColor,attr"`
	StrokeWidth float64 `xml:"strokeWidth,attr"`
	PathData    string  `xml:"pathData,attr"`
}

type colorDef struct {
	Name  string `xml:"name,attr"`
	Color string `xml:",chardata"`
}

type colorDefsArray struct {
	Colors []colorDef `xml:"color"`
}

type colorDefs map[string]color.Color

func (cd *colorDefs) String() string {
	return ""
}

func (cd *colorDefs) Set(value string) error {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("Invalid color definition \"%s\"", value)
	}
	color, err := parseColor(strings.TrimSpace(parts[1]), nil)
	if err != nil {
		return fmt.Errorf("Invalid color definition \"%s\"", value)
	}

	name := strings.TrimSpace(parts[0])
	(*cd)[name] = color
	return nil
}

func main() {
	colorDefs := make(colorDefs)
	colorsFile := ""
	ios := false
	scaleFactor := 1.0
	showVersion := false
	vectorFile := ""
	pngFile := ""

	flag.Var(&colorDefs, "color", "Defines an (A)RGB value for a color name (name=#(a)rgb|(aa)rrggbb)")
	flag.StringVar(&colorsFile, "colors", colorsFile, "Defines an Android color resource file to be parsed for color definitions")
	flag.Float64Var(&scaleFactor, "scale", scaleFactor, "Scales the image by the given factor")
	flag.BoolVar(&ios, "ios", ios, "Generates three resolutions of the image (adds @2x and @3x versions)")
	flag.BoolVar(&showVersion, "version", false, "Shows the program version")
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options] <vector-image-input> [<png-image-output>]\n\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
		fmt.Println()
	}
	flag.Parse()

	if showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if flag.NArg() == 1 {
		vectorFile = flag.Arg(0)
		pngFile = pathWithoutExtension(vectorFile) + ".png"
	} else if flag.NArg() == 2 {
		vectorFile = flag.Arg(0)
		pngFile = flag.Arg(1)
	} else {
		fmt.Println("ERROR: Input vector image parameter is missing")
		flag.Usage()
		os.Exit(1)
	}

	if colorsFile != "" {
		parseColorsFile(colorsFile, &colorDefs)
	}

	xmlData, err := os.ReadFile(vectorFile)
	if err != nil {
		errorExit("Cannot read vector file", err)
	}

	var vec vector
	if err := xml.Unmarshal(xmlData, &vec); err != nil {
		errorExit("Cannot parse vector file", err)
	} else if vec.XMLName.Local != "vector" {
		errorExit("Not a valid Android vector drawable", nil)
	}

	c, err := renderVector(&vec, colorDefs)
	if err != nil {
		errorExit("Cannot render vector file", err)
	}

	saveCanvas(c, pngFile, scaleFactor)
	if ios {
		saveCanvas(c, pathWithoutExtension(pngFile)+"@2x.png", 2*scaleFactor)
		saveCanvas(c, pathWithoutExtension(pngFile)+"@3x.png", 3*scaleFactor)
	}
}

func renderVector(vec *vector, colorDefs colorDefs) (*canvas.Canvas, error) {
	width, err := parseDpNum(vec.Width, "width")
	if err != nil {
		return nil, err
	}

	height, err := parseDpNum(vec.Height, "height")
	if err != nil {
		return nil, err
	}

	c := canvas.New(width, height)
	ctx := canvas.NewContext(c)
	ctx.SetCoordSystem(canvas.CartesianIV)
	ctx.SetView(canvas.Identity.Scale(width/vec.ViewportWidth, height/vec.ViewportHeight))

	for _, pathElem := range vec.Paths {
		path := canvas.MustParseSVGPath(pathElem.PathData)
		ctx.SetFillColor(canvas.Transparent)
		ctx.SetStrokeColor(canvas.Transparent)
		ctx.SetStrokeWidth(pathElem.StrokeWidth)
		if pathElem.FillColor != "" {
			c, err := parseColor(pathElem.FillColor, colorDefs)
			if err != nil {
				return nil, err
			}
			ctx.SetFillColor(c)
		}
		if pathElem.StrokeColor != "" {
			c, err := parseColor(pathElem.StrokeColor, colorDefs)
			if err != nil {
				return nil, err
			}
			ctx.SetStrokeColor(c)
		}
		ctx.DrawPath(0, 0, path)
	}

	return c, nil
}

func saveCanvas(c *canvas.Canvas, p string, scaleFactor float64) {
	err := renderers.Write(p, c, canvas.DPMM(scaleFactor))
	if err != nil {
		errorExit(fmt.Sprintf("Cannot save PNG data to \"%s\"", p), err)
	}
}

func parseColorsFile(colorsFile string, colorDefs *colorDefs) {
	colorsData, err := os.ReadFile(colorsFile)
	if err != nil {
		errorExit("Cannot read colors file", err)
	}
	var colorsArray colorDefsArray
	err = xml.Unmarshal(colorsData, &colorsArray)
	if err != nil {
		errorExit("Cannot parse colors file", err)
	}

	colors := colorsArray.Colors
	var remainingColors []colorDef
	for true {
		for _, colorDef := range colors {
			color, err := parseColor(colorDef.Color, *colorDefs)
			if err == nil {
				(*colorDefs)["@color/"+colorDef.Name] = color
			} else {
				remainingColors = append(remainingColors, colorDef)
			}
		}
		if len(remainingColors) == 0 || len(colors) == len(remainingColors) {
			break
		}
		colors = remainingColors
	}
}

func parseColor(c string, colorDefs colorDefs) (color.Color, error) {
	if color, ok := colorDefs[c]; ok {
		return color, nil
	}

	match := colorPattern.FindSubmatch([]byte(c))
	if match == nil {
		return nil, fmt.Errorf("Invalid color \"%s\"", c)
	}

	spec := match[1]
	if len(spec) == 3 {
		r := hexToValue(spec[0])
		g := hexToValue(spec[1])
		b := hexToValue(spec[2])
		return color.NRGBA{r<<4 | r, g<<4 | g, b<<4 | b, 255}, nil
	} else if len(spec) == 4 {
		a := hexToValue(spec[0])
		r := hexToValue(spec[1])
		g := hexToValue(spec[2])
		b := hexToValue(spec[3])
		return color.NRGBA{r<<4 | r, g<<4 | g, b<<4 | b, a<<4 | a}, nil
	} else if len(spec) == 6 {
		r1 := hexToValue(spec[0])
		r2 := hexToValue(spec[1])
		g1 := hexToValue(spec[2])
		g2 := hexToValue(spec[3])
		b1 := hexToValue(spec[4])
		b2 := hexToValue(spec[5])
		return color.NRGBA{r1<<4 | r2, g1<<4 | g2, b1<<4 | b2, 255}, nil
	} else if len(spec) == 8 {
		a1 := hexToValue(spec[0])
		a2 := hexToValue(spec[1])
		r1 := hexToValue(spec[2])
		r2 := hexToValue(spec[3])
		g1 := hexToValue(spec[4])
		g2 := hexToValue(spec[5])
		b1 := hexToValue(spec[6])
		b2 := hexToValue(spec[7])
		return color.NRGBA{r1<<4 | r2, g1<<4 | g2, b1<<4 | b2, a1<<4 | a2}, nil
	} else {
		return nil, fmt.Errorf("Invalid color \"%s\"", c)
	}
}

func parseDpNum(n string, name string) (float64, error) {
	match := dpNumPattern.FindStringSubmatch(n)
	if match == nil {
		return 0, fmt.Errorf("Invalid %s \"%s\"", name, n)
	}

	width, err := strconv.ParseFloat(match[1], 32)
	if err != nil {
		return 0, err
	}
	return width, nil
}

func hexToValue(n byte) uint8 {
	if n >= '0' && n <= '9' {
		return n - '0'
	} else if n >= 'a' && n <= 'f' {
		return n - 'a' + 10
	} else if n >= 'A' && n <= 'F' {
		return n - 'A' + 10
	}
	return 0
}

func pathWithoutExtension(p string) string {
	return strings.TrimSuffix(p, filepath.Ext(p))
}

func errorExit(msg string, err error) {
	fmt.Print("ERROR: ")
	fmt.Print(msg)
	if err != nil {
		fmt.Print(" (")
		fmt.Print(err)
		fmt.Print(")")
	}
	fmt.Println()
	os.Exit(1)
}
