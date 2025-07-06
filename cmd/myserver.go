/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"

	"github.com/atharvwasthere/Fastlane/server"
	"github.com/spf13/cobra"
)

// myserverCmd represents the myserver command
var myserverCmd = &cobra.Command{
	Use:   "myserver",
	Short: "Display detailed info about your nearest test server",
	Long: `myserver locates the nearest speed test server based on your IP-derived location and 
		   displays detailed information such as server name, host, country, sponsor, and distance. 
		   It uses Ookla APIs or the GeoLite2 database to determine proximity.`,
		   
	Run: func(cmd *cobra.Command, args []string) {
		err := RunMyServer(cmd, args) 
		if err != nil {
		fmt.Println("Download failed:", err)
		}
	},
}

func RunMyServer(cmd *cobra.Command, args []string) error {
	selector, _ := server.NewSelector(context.Background(),"assets/servers.json", nil, "assets/GeoLite2-City.mmdb")
	server, err := selector.MyServer()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return 
	}
	fmt.Printf("Server: %s\nHost: %s\nCountry: %s\nLocation: %s\nSponsor: %s\n",
        server.Name, server.Host, server.Country, server.Location, server.Sponsor)
}

func init() {
	rootCmd.AddCommand(myserverCmd)


	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// myserverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// myserverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
