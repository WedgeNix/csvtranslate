package csvtranslate

import (
	"log"
	"os"

	"encoding/csv"

	"sync"

	"fmt"

	"errors"

	"cloud.google.com/go/translate"
	"golang.org/x/net/context"
	"golang.org/x/text/language"
	"google.golang.org/api/option"
)

var (
	apiKey string
	wg     sync.WaitGroup
)

// ExcelTrans holding  data needed to translate file.
type ExcelTrans struct {
	source language.Tag
	target language.Tag
	toDir  string
	file   *os.File
}

// New creates controler to start translating excel files.
// Takes file directory with file name, and a language tag.
// The api key must be stored in envionmental vars under
// TRANSLATE_API_KEY.
func New(fileDir string, source language.Tag) (*ExcelTrans, error) {
	apiKey = os.Getenv("TRANSLATE_API_KEY")

	f, err := os.Open(fileDir)
	// defer f.Close()
	if err != nil {
		return nil, err
	}
	return &ExcelTrans{
		source: source,
		file:   f,
	}, nil
}

// SetToDirectory sets to to directory if not default is directory of go program.ss
func (ext *ExcelTrans) SetToDirectory(dir string) {
	ext.toDir = dir
}

// SetTarget sets target language to translate to.
func (ext *ExcelTrans) SetTarget(target language.Tag) {
	ext.target = target
}

type transPair struct {
	spot      int
	tranlated []string
	errors    error
}

// TranslateCSV translates CSV file into target language.
// Takes column numbers to translate.
func (ext *ExcelTrans) TranslateCSV(cols ...int) error {
	if ext.toDir == "" {
		return errors.New("Use SetToDirectory()")
	}
	defer ext.file.Close()
	transCols := make(chan transPair)
	done := make(chan bool)
	csvReader := csv.NewReader(ext.file)
	csvR, err := csvReader.ReadAll()
	if err != nil {
		return err
	}

	trans, err := newTranslate(ext.source)
	if err != nil {
		return err
	}
	wg.Add(len(cols))
	go func() {
		wg.Wait()
		done <- true
	}()

	for _, col := range cols {
		go func(csvr [][]string, c int, tp chan transPair) {
			defer wg.Done()
			myCol := []string{}
			for _, row := range csvr {
				myCol = append(myCol, row[c-1])
			}
			trans, err := trans.translate(myCol, ext.target)

			tp <- transPair{c - 1, trans, err}

		}(csvR, col, transCols)
	}

	for {
		select {
		case t := <-transCols:
			for i, j := range t.tranlated {
				csvR[i][t.spot] = j
			}
		case <-done:
			fName := ext.toDir
			fmt.Println(fName)
			file, err := os.Create(fName)
			if err != nil {
				log.Panicln(err)
			}
			defer file.Close()

			writer := csv.NewWriter(file)
			defer writer.Flush()

			writer.WriteAll(csvR)
			return nil
		}
	}

}

/************************************************************
* Google translate code to translate data to other language.*
* Used TransKU project from WedgeNix.                       *
*************************************************************
 */

// TransKU holds the library data.
type transKU struct {
	context context.Context
	client  *translate.Client
	options *translate.Options
}

// New creates the base TransKU instance.
func newTranslate(source language.Tag) (*transKU, error) {
	context := context.Background()
	client, err := translate.NewClient(context, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	return &transKU{
		context,
		client,
		&translate.Options{Source: source},
	}, err
}

// Source sets the new source language for the translator.
func (t *transKU) source(source language.Tag) {
	t.options.Source = source
}

// Translate uses the current source language and translates input based on a
// target language.
func (t *transKU) translate(inputs []string, target language.Tag) ([]string, error) {
	// defer wg.Done()
	trans, err := t.client.Translate(t.context, inputs, target, t.options)
	if err != nil {
		return nil, err
	}
	lang := []string{}
	for _, t := range trans {
		lang = append(lang, t.Text)
	}

	return lang, nil

}
