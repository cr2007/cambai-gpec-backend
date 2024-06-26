package camb

import (
	"fmt"
	"os"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
)

type Camb struct {
	URL          string
	APIKey       string
	ResendAPIKey string
}

func Init() Camb {
	return Camb{
		URL:          "https://client.camb.ai/apis",
		APIKey:       os.Getenv("CAMB_API_KEY"),
		ResendAPIKey: os.Getenv("RESEND_API_KEY"),
	}
}

func (c *Camb) API(endpoint string) string {
	return fmt.Sprintf("%s%s", c.URL, endpoint)
}

func (c *Camb) StartDubbingPipeline(
	app *pocketbase.PocketBase,
	record *models.Record,
	email string,
	userName string,
	VideoURL string,
) {
	dubbingResp, err := c.StartDubbing(StartDubbingRequestBody{
		VideoURL:       VideoURL,
		SourceLanguage: record.GetInt("source_id"),
		TargetLanguage: record.GetInt("target_id"),
	})
	if err != nil || dubbingResp.TaskID == "" {
		fmt.Println(err.Error(), dubbingResp)
		record.Set("status", "Upload to CAMB.AI servers failed!")
		app.Dao().SaveRecord(record)
		return
	}
	record.Set("status", "Uploaded to CAMB.AI servers")
	record.Set("task_id", dubbingResp.TaskID)

	app.Dao().SaveRecord(record)

	var statusResp StatusResponse

	foundRunID := false

	for {

		statusResp, err = c.DubbingStatus(dubbingResp)
		if err != nil || (statusResp.Status != "SUCCESS" && statusResp.Status != "PENDING") {
			record.Set("status", "Error getting task status")
			app.Dao().SaveRecord(record)
			return
		} else if (statusResp.RunID != 0) && !foundRunID {
			record.Set("status", "The video is being dubbed, once complete will be sent to "+email)
			record.Set("run_id", statusResp.RunID)
			foundRunID = true
			app.Dao().SaveRecord(record)
		}

		fmt.Println("Polling status:", statusResp)
		if statusResp.Status == "SUCCESS" {
			record.Set("status", "Dubbing Complete!")
			break
		}

		time.Sleep(1 * time.Second)
	}

	c.SendEmail(app, email, statusResp, record, userName)
}
