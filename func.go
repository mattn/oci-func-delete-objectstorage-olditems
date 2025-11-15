package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	fdk "github.com/fnproject/fdk-go"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/common/auth"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
)

type config struct {
	BucketName    string `json:"bucket-name"`
	RetentionDays int    `json:"retention-days"`
}

func main() {
	fdk.Handle(fdk.HandlerFunc(func(ctx context.Context, in io.Reader, out io.Writer) {
		days, err := strconv.Atoi(os.Getenv("RETENTION_DAYS"))
		if err != nil {
			days = 30
		}

		cfg := config{
			BucketName:    os.Getenv("BUCKET_NAME"),
			RetentionDays: days,
		}
		if json.NewDecoder(in).Decode(&cfg) != nil {
			if cfg.BucketName == "" {
				fmt.Fprintln(out, "BUCKET_NAME environment variable is required")
				fdk.WriteStatus(out, http.StatusInternalServerError)
				return
			}
		}

		configurationProvider, err := auth.ResourcePrincipalConfigurationProvider()
		if err != nil {
			fmt.Fprintf(out, "RP provider error: %v\n", err)
			fdk.WriteStatus(out, http.StatusInternalServerError)
			return
		}

		c, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(configurationProvider)
		if err != nil {
			fmt.Fprintf(out, "client error: %v\n", err)
			fdk.WriteStatus(out, http.StatusInternalServerError)
			return
		}

		nsResp, err := c.GetNamespace(ctx, objectstorage.GetNamespaceRequest{})
		if err != nil {
			fmt.Fprintf(out, "namespace error: %v\n", err)
			fdk.WriteStatus(out, http.StatusInternalServerError)
			return
		}

		namespace := *nsResp.Value
		now := time.Now().Add(-time.Duration(cfg.RetentionDays) * 24 * time.Hour)

		deletedCount := 0

		var deleteOldObjects func(prefix string)
		deleteOldObjects = func(prefix string) {
			resp, err := c.ListObjects(ctx, objectstorage.ListObjectsRequest{
				NamespaceName: common.String(namespace),
				BucketName:    common.String(cfg.BucketName),
				Prefix:        &prefix,
				Delimiter:     common.String("/"),
				Fields:        common.String("name,timeCreated"),
			})
			if err != nil {
				fmt.Fprintf(out, "ListObjects ERROR prefix=%q: %v\n", prefix, err)
				return
			}
			for _, obj := range resp.Objects {
				if obj.TimeCreated != nil && obj.TimeCreated.Time.Before(now) {
					fmt.Fprintf(out, "deleting: %s (created: %v)\n", *obj.Name, obj.TimeCreated)

					_, err := c.DeleteObject(ctx, objectstorage.DeleteObjectRequest{
						NamespaceName: common.String(namespace),
						BucketName:    common.String(cfg.BucketName),
						ObjectName:    obj.Name,
						IfMatch:       obj.Etag,
					})
					if err == nil {
						deletedCount++
					}
				}
			}
			for _, p := range resp.Prefixes {
				sub := p
				if !strings.HasSuffix(sub, "/") {
					sub += "/"
				}
				deleteOldObjects(sub)
			}
		}

		deleteOldObjects("")

		fmt.Fprintf(out, "Total deleted objects: %d (bucket: %s, retention: %d days)\n", deletedCount, cfg.BucketName, cfg.RetentionDays)
	}))
}
