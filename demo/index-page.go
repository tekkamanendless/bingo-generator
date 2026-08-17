package demo

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strings"

	"github.com/go-app-blazar/blazar/blazar"
	"github.com/maxence-charriere/go-app/v11/pkg/app"
)

type IndexPage struct {
	app.Compo

	count     uint
	size      uint
	targetURL string

	bingoCards []BingoCard
}

type BingoCard struct {
	Rows [][]string
}

func (c *BingoCard) String() string {
	return fmt.Sprintf("%+v", *c)
}

func (c *IndexPage) OnMount(ctx app.Context) {
	slog.InfoContext(ctx.Context, "IndexPage: OnMount")

	c.targetURL = ""
	c.count = 5
	c.size = 5
}

func (c *IndexPage) OnNav(ctx app.Context) {
	slog.InfoContext(ctx.Context, "IndexPage: OnNav")
}

func (c *IndexPage) Render() app.UI {
	return blazar.Page().
		Body(
			blazar.Collapse().
				Label("Configuration").
				Open(true).
				Body(
					blazar.Input[string]().
						Label("Target URL").
						Bind(&c.targetURL),
					blazar.Input[uint]().
						Label("Grid Size").
						Bind(&c.size),
					blazar.Input[uint]().
						Label("Count").
						Bind(&c.count),
				),
			blazar.Form().
				Action(blazar.FormAction{
					Name:     "Generate",
					Function: c.generateBingoCards,
				}),
			app.If(len(c.bingoCards) > 0, func() app.UI {
				return app.Div().
					Class("bingo-cards").
					Body(
						app.Range(c.bingoCards).Slice(func(i int) app.UI {
							card := c.bingoCards[i]

							uiCells := []app.UI{}
							cellCount := 0
							for _, row := range card.Rows {
								for _, cell := range row {
									uiCells = append(uiCells, app.Div().
										Class("bingo-card-cell").
										Text(cell))
									cellCount++
								}
							}

							return app.Div().
								Body(
									app.H2().Text("Bingo"),
									app.Div().
										Class("bingo-card").
										Style("grid-template-columns", fmt.Sprintf("repeat(%d, 1fr)", c.size)).
										Style("grid-template-rows", fmt.Sprintf("repeat(%d, 1fr)", c.size)).
										Body(
											uiCells...,
										),
								)
						}),
					)
			}),
		)
}

func (c *IndexPage) generateBingoCards(ctx app.Context) {
	slog.InfoContext(ctx.Context, "IndexPage: generateBingoCards")

	options, err := c.fetchOptions(ctx.Context)
	if err != nil {
		slog.ErrorContext(ctx.Context, "IndexPage: generateBingoCards", "error", err)
		return
	}

	numberOfCells := c.size * c.size
	if len(options) < int(numberOfCells) {
		slog.ErrorContext(ctx.Context, "IndexPage: generateBingoCards", "error", "not enough options")
		return
	}

	c.bingoCards = []BingoCard{}
	usedCards := map[string]bool{}
	for range c.count {
		bingoCard := BingoCard{
			Rows: [][]string{},
		}

		cardOptions := make([]string, len(options))
		copy(cardOptions, options)
		rand.Shuffle(len(cardOptions), func(i, j int) {
			cardOptions[i], cardOptions[j] = cardOptions[j], cardOptions[i]
		})

		if len(cardOptions) > int(numberOfCells) {
			cardOptions = cardOptions[:numberOfCells]
		}

		i := 0
		for range c.size {
			row := []string{}
			for range c.size {
				row = append(row, cardOptions[i])
				i++
			}
			bingoCard.Rows = append(bingoCard.Rows, row)
		}
		usedCards[bingoCard.String()] = true
		c.bingoCards = append(c.bingoCards, bingoCard)
	}

	ctx.Update()
}

func (c *IndexPage) fetchOptions(ctx context.Context) ([]string, error) {
	slog.InfoContext(ctx, "IndexPage: fetchOptions")

	if c.targetURL == "" {
		return nil, errors.New("target URL is required")
	}

	resp, err := http.Get(c.targetURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		return nil, errors.New("content type is required")
	}

	contentType = strings.Split(contentType, ";")[0]
	switch contentType {
	case "text/csv":
		csvReader := csv.NewReader(bytes.NewReader(body))
		rows, err := csvReader.ReadAll()
		if err != nil {
			return nil, err
		}
		options := []string{}
		for _, row := range rows {
			options = append(options, row[0])
		}
		return options, nil
	default:
		return nil, errors.New("content type is not supported")
	}
}
