package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/google/generative-ai-go/genai"
	"github.com/pelletier/go-toml/v2"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type Config struct {
	Model struct {
		Name string `toml:"name"`
	} `toml:"model"`
}

type RequestData struct {
	Input       string `json:"input"`
	SystemInfo  string `json:"system_info"`
	HistoryFile string `json:"history_file"`
	Packages    string `json:"packages"`
}

var config Config

func ReadConfig() {
	data, err := os.ReadFile("config/shell.toml")
	if err != nil {
		log.Fatalf("Error reading TOML file: %v", err)
	}

	err = toml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Error unmarshaling TOML: %v", err)
	}
}

func main() {
	ReadConfig()

	app := fiber.New()

	app.Post("/", func(c fiber.Ctx) error {
		ctx := context.Background()
		client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINIAPIKEY")))
		if err != nil {
			return err
		}
		defer client.Close()

		var requestData RequestData
		if err := c.Bind().Body(&requestData); err != nil { // Corrected line
			return c.Status(http.StatusBadRequest).SendString("Error parsing request body: " + err.Error())
		}

		model := client.GenerativeModel(config.Model.Name)
		fi, err := os.ReadFile("shell.md")
		if err != nil {
			return err
		}

		model.SystemInstruction = genai.NewUserContent(genai.Text(string(fi)))

		parts := []genai.Part{}
		if requestData.SystemInfo != "" {
			parts = append(parts, genai.Text("System Info:\n"+requestData.SystemInfo))
		}
		if requestData.HistoryFile != "" {
			parts = append(parts, genai.Text("History File:\n"+requestData.HistoryFile))
		}
		if requestData.Packages != "" {
			parts = append(parts, genai.Text("Packages:\n"+requestData.Packages))
		}
		if requestData.Input != "" {
			parts = append(parts, genai.Text(requestData.Input))
		}

		iter := model.GenerateContentStream(ctx, parts...)

		c.Set("Content-Type", "text/plain; charset=utf-8")
		c.Set("Transfer-Encoding", "chunked")

		return c.SendStreamWriter(func(w *bufio.Writer) {
			defer w.Flush()

			for {
				resp, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					log.Printf("Stream error: %v\n", err)
					fmt.Fprintf(w, "Error: %v\n", err) // Write to the buffered writer
					return
				}

				for _, cand := range resp.Candidates {
					if cand.Content != nil {
						for _, part := range cand.Content.Parts {
							if textPart, ok := part.(genai.Text); ok {
								_, writeErr := fmt.Fprint(w, textPart)
								if writeErr != nil {
									log.Printf("Write error: %v\n", writeErr)
									return
								}
							}
						}
					}
				}
			}
		})
	})

	log.Fatal(app.Listen(":8000"))
}
