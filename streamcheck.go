package main

import (
	"html"
	"fmt"
	"context"
	"log"
	"strings"
	"os"
	"bufio"
	"io"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)


func LookForPremiere(service *youtube.Service, channelId string) ([]*youtube.SearchResult, error) {
	call := service.Search.List([]string{"id,snippet"}).
        Type("video").
        EventType("upcoming").
        ChannelId(channelId).
        MaxResults(4)

    response, err := call.Do()
    if err != nil {
        return nil, err
    }

    return response.Items, nil
}

/*
func LookForLive(service *youtube.Service, channelId string) ([]*youtube.SearchResult, error) {
	call := service.Search.List([]string{"id,snippet"}).
        Type("video").
        EventType("live").
        ChannelId(channelId).
        MaxResults(1)

    response, err := call.Do()
    if err != nil {
        return nil, err
    }

    return response.Items, nil
}
*/
/*
func GetChannelID(service *youtube.Service, username string) (string, error) {
    call := service.Channels.List([]string{"id"}).
        ForUsername(username).
        MaxResults(1)

    response, err := call.Do()
    if err != nil {
        return "", err
    }

    if len(response.Items) == 0 {
        return "", fmt.Errorf("no channel found for username %s", username)
    }

    return response.Items[0].Id, nil
}
*/

func main() {
	file, err := os.Open("apikey")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	apikey, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading API key:", err)
		return
	}

	channels, err := os.Open("commented-id.txt")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer channels.Close()

	ctx := context.Background()
	youtubeService, err := youtube.NewService(ctx, option.WithAPIKey(apikey))

	streamerRead := bufio.NewReader(channels)
	for {
		channelBytes, err := streamerRead.ReadBytes(' ')
		untidyChannel := string(channelBytes)
		channelid := strings.TrimRight(untidyChannel,"\n")
		if err != nil && err != io.EOF {
			fmt.Println("Error reading channel id:", err)
			return
		}
		/* skip everything until our comment */
		_, err = streamerRead.ReadBytes('#')
		if err != nil {
			if err == io.EOF {
			/* no comment found */
				break
			}
			fmt.Println("Error finding delimiter:", err)
			return
		}
		/* read our comment until newline */
		streamerBytes, err := streamerRead.ReadBytes('\n')
		if err != nil && err != io.EOF {
			fmt.Println("Error reading comment:", err)
			return
		}
		untidyStreamer := string(streamerBytes)
		streamer := strings.TrimRight(untidyStreamer,"\n")
		fmt.Printf("%s's upcoming or currently live stream(s):\n", streamer)
		results, err := LookForPremiere(youtubeService, channelid)
		if err != nil {
			log.Fatalf("Error searching for channel premieres: %v", err)
		}
		for _, item := range results {
			stream, err := youtubeService.Videos.List([]string{"LiveStreamingDetails"}).Id(item.Id.VideoId).Do()
			if err != nil {
				fmt.Printf("Error fetching video details: %v\n", err)
				continue
			}
			startTime := "" /* initialize before the if or errors out */
			if len(stream.Items) > 0 {
				startTime = stream.Items[0].LiveStreamingDetails.ScheduledStartTime
			} else {
				fmt.Printf("Video details not found for ID: %v\n", item.Id.VideoId)
			}
			startTimeParse, err := time.Parse(time.RFC3339, startTime)
			if err != nil {
				fmt.Printf("Error parsing startTime: %v\n", err)
				return
			}

			/* convert UTC to EST */
			estLocation, err := time.LoadLocation("America/New_York")
			if err != nil {
				fmt.Printf("Error loading location: %v\n", err)
				return
			}
			estTime := startTimeParse.In(estLocation)
			
			/* check if the time is within DST */
			isDST := estTime.Year() == estTime.In(estLocation).AddDate(0, 0, -1).Year()

			/* format the time appropriate time zone abbreviation */
			var tzAbbreviation string
			if isDST {
				tzAbbreviation = "EDT"
			} else {
				tzAbbreviation = "EST"
			}
			/* you have to give it a format to go off of */
			formattedTime := estTime.Format("01-02-06 3:04 PM ") + tzAbbreviation

			livecheck := item.Snippet.LiveBroadcastContent
			fmt.Printf("%v - Starts on: %v - %s\n", html.UnescapeString(item.Snippet.Title), formattedTime, livecheck)
		}
		/*
		live_results, err := LookForLive(youtubeService, channelid)
		if err != nil {
			log.Fatalf("Error searching for channel streams: %v", err)
		}
		fmt.Printf("%s's currently live stream:\n", streamer)
		for _, item := range live_results {
			fmt.Printf("%v\n", html.UnescapeString(item.Snippet.Title))
		}
		*/
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}
