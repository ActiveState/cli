package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ActiveState/cli/internal/chanutils/workerpool"
	"github.com/ActiveState/cli/internal/errs"
)

func LookupCve(cveIds ...string) (map[string]interface{}, error) {
	results := map[string]interface{}{}
	// https://api.osv.dev/v1/vulns/OSV-2020-111
	wp := workerpool.New(5)
	for _, cveId := range cveIds {
		wp.Submit(func() error {
			resp, err := http.Get(fmt.Sprintf("https://api.osv.dev/v1/vulns/%s", cveId))
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return err
			}
			results[cveId] = result
			return nil
		})
	}

	err := wp.Wait()
	if err != nil {
		return nil, errs.Wrap(err, "Failed to wait for workerpool")
	}

	return results, nil
}