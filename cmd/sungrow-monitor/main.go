package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"sungrow-monitor/config"
	"sungrow-monitor/internal/api"
	"sungrow-monitor/internal/collector"
	"sungrow-monitor/internal/inverter"
	"sungrow-monitor/internal/modbus"
	"sungrow-monitor/internal/mqtt"
	"sungrow-monitor/internal/storage"

	"github.com/spf13/cobra"
)

var (
	configFile string
	verbose    bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "sungrow-monitor",
		Short: "Sungrow inverter monitor",
		Long:  "A tool to monitor Sungrow SG5.0RS-S inverter via Modbus TCP",
	}

	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(serveCmd())
	rootCmd.AddCommand(readCmd())
	rootCmd.AddCommand(testCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the monitoring service",
		Long:  "Start the collector, API server, and MQTT publisher",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create Modbus client
			modbusClient := modbus.NewClient(
				cfg.Inverter.IP,
				cfg.Inverter.Port,
				cfg.Inverter.SlaveID,
				cfg.Inverter.Timeout,
			)

			// Create database
			db, err := storage.NewDatabase(cfg.Database.Path)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			log.Printf("Database opened at %s", cfg.Database.Path)

			// Create MQTT publisher
			publisher, err := mqtt.NewPublisher(mqtt.PublisherConfig{
				Broker:      cfg.MQTT.Broker,
				ClientID:    cfg.MQTT.ClientID,
				Username:    cfg.MQTT.Username,
				Password:    cfg.MQTT.Password,
				TopicPrefix: cfg.MQTT.TopicPrefix,
				Enabled:     cfg.MQTT.Enabled,
			})
			if err != nil {
				log.Printf("Warning: MQTT connection failed: %v", err)
			} else if cfg.MQTT.Enabled {
				log.Printf("MQTT connected to %s", cfg.MQTT.Broker)
				// Publish Home Assistant discovery
				publisher.PublishHomeAssistantDiscovery()
			}

			// Create collector
			coll := collector.NewCollector(collector.CollectorConfig{
				Client:    modbusClient,
				Database:  db,
				Publisher: publisher,
				Interval:  cfg.Collector.Interval,
				Enabled:   cfg.Collector.Enabled,
			})

			// Setup context for graceful shutdown
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Handle signals
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			// Start collector in goroutine
			go func() {
				if err := coll.Start(ctx); err != nil {
					log.Printf("Collector error: %v", err)
				}
			}()

			// Start API server if enabled
			if cfg.API.Enabled {
				server := api.NewServer(api.ServerConfig{
					Port:      cfg.API.Port,
					Collector: coll,
					Database:  db,
				})

				go func() {
					if err := server.Start(); err != nil {
						log.Printf("API server error: %v", err)
					}
				}()
			}

			log.Println("Sungrow Monitor started. Press Ctrl+C to stop.")

			// Wait for signal
			<-sigChan
			log.Println("Shutting down...")
			cancel()
			coll.Stop()

			return nil
		},
	}
}

func readCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "read",
		Short: "Read data once from the inverter",
		Long:  "Connect to the inverter and read all data once",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			client := modbus.NewClient(
				cfg.Inverter.IP,
				cfg.Inverter.Port,
				cfg.Inverter.SlaveID,
				cfg.Inverter.Timeout,
			)

			if err := client.Connect(); err != nil {
				return fmt.Errorf("failed to connect: %w", err)
			}
			defer client.Close()

			sungrow := inverter.NewSungrow(client)
			data, err := sungrow.ReadAllData()
			if err != nil {
				return fmt.Errorf("failed to read data: %w", err)
			}

			output, _ := json.MarshalIndent(data, "", "  ")
			fmt.Println(string(output))

			return nil
		},
	}
}

func testCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Test connection to the inverter",
		Long:  "Test the Modbus TCP connection to the inverter",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			fmt.Printf("Testing connection to %s:%d...\n", cfg.Inverter.IP, cfg.Inverter.Port)

			client := modbus.NewClient(
				cfg.Inverter.IP,
				cfg.Inverter.Port,
				cfg.Inverter.SlaveID,
				cfg.Inverter.Timeout,
			)

			sungrow := inverter.NewSungrow(client)
			if err := sungrow.TestConnection(); err != nil {
				fmt.Printf("Connection FAILED: %v\n", err)
				return err
			}

			fmt.Println("Connection SUCCESS!")

			// Read and display basic info
			data, err := sungrow.ReadAllData()
			if err != nil {
				fmt.Printf("Warning: Could not read data: %v\n", err)
			} else {
				fmt.Printf("\nInverter Info:\n")
				fmt.Printf("  Serial Number: %s\n", data.SerialNumber)
				fmt.Printf("  Device Type:   %d\n", data.DeviceTypeCode)
				fmt.Printf("  Nominal Power: %.1f kW\n", data.NominalPower)
				fmt.Printf("  Output Type:   %s\n", data.OutputType)
				fmt.Printf("  Status:        %s\n", data.RunningStateString)
				fmt.Printf("\nCurrent Values:\n")
				fmt.Printf("  Power:         %d W\n", data.TotalActivePower)
				fmt.Printf("  Daily Energy:  %.1f kWh\n", data.DailyEnergy)
				fmt.Printf("  Total Energy:  %.1f kWh\n", data.TotalEnergy)
				fmt.Printf("  Temperature:   %.1f °C\n", data.Temperature)
			}

			client.Close()
			return nil
		},
	}
}
