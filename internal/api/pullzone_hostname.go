// Copyright (c) BunnyWay d.o.o.
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type PullzoneHostname struct {
	Id               int64  `json:"Id,omitempty"`
	PullzoneId       int64  `json:"-"`
	Name             string `json:"Value"`
	IsSystemHostname bool   `json:"IsSystemHostname"`
	HasCertificate   bool   `json:"HasCertificate"`
	ForceSSL         bool   `json:"ForceSSL"`
}

func (c *Client) CreatePullzoneHostname(data PullzoneHostname) (PullzoneHostname, error) {
	pullzoneId := data.PullzoneId
	if pullzoneId == 0 {
		return PullzoneHostname{}, errors.New("pullzone is required")
	}

	body, err := json.Marshal(map[string]string{
		"Hostname": data.Name,
	})

	if err != nil {
		return PullzoneHostname{}, err
	}

	resp, err := c.doRequest(http.MethodPost, fmt.Sprintf("%s/pullzone/%d/addHostname", c.apiUrl, pullzoneId), bytes.NewReader(body))
	if err != nil {
		return PullzoneHostname{}, err
	}

	if resp.StatusCode != http.StatusNoContent {
		return PullzoneHostname{}, errors.New("addHostname failed with " + resp.Status)
	}

	pullzone, err := c.GetPullzone(pullzoneId)
	if err != nil {
		return PullzoneHostname{}, err
	}

	for _, hostname := range pullzone.Hostnames {
		if hostname.Name == data.Name {
			previousData := PullzoneHostname{HasCertificate: hostname.HasCertificate}
			hostname.PullzoneId = pullzoneId
			hostname.HasCertificate = data.HasCertificate
			hostname.ForceSSL = data.ForceSSL

			return c.UpdatePullzoneHostname(hostname, previousData)
		}
	}

	return PullzoneHostname{}, errors.New("Hostname not found")
}

func (c *Client) UpdatePullzoneHostname(data PullzoneHostname, previousData PullzoneHostname) (PullzoneHostname, error) {
	pullzoneId := data.PullzoneId
	if pullzoneId == 0 {
		return PullzoneHostname{}, errors.New("pullzone is required")
	}

	if data.IsSystemHostname && previousData.HasCertificate && !data.HasCertificate {
		return PullzoneHostname{}, errors.New("removing a certificate from an internal hostname is not supported")
	}

	// remove existing certificate
	if !data.IsSystemHostname && previousData.HasCertificate && !data.HasCertificate {
		body, err := json.Marshal(map[string]interface{}{
			"Hostname": data.Name,
		})

		if err != nil {
			return PullzoneHostname{}, err
		}

		resp, err := c.doRequest(http.MethodDelete, fmt.Sprintf("%s/pullzone/%d/removeCertificate", c.apiUrl, pullzoneId), bytes.NewReader(body))
		if err != nil {
			return PullzoneHostname{}, err
		}

		if resp.StatusCode != http.StatusNoContent {
			return PullzoneHostname{}, errors.New("removeCertificate failed with " + resp.Status)
		}
	}

	// manage free TLS certificate
	if !data.IsSystemHostname && !previousData.HasCertificate && data.HasCertificate {
		resp, err := c.doRequest(http.MethodGet, fmt.Sprintf("%s/pullzone/loadFreeCertificate?hostname=%s", c.apiUrl, data.Name), nil)
		if err != nil {
			return PullzoneHostname{}, err
		}

		if resp.StatusCode != http.StatusOK {
			return PullzoneHostname{}, errors.New("loadFreeCertificate failed with " + resp.Status)
		}
	}

	// force SSL
	{
		body, err := json.Marshal(map[string]interface{}{
			"ForceSSL": data.ForceSSL,
			"Hostname": data.Name,
		})

		if err != nil {
			return PullzoneHostname{}, err
		}

		resp, err := c.doRequest(http.MethodPost, fmt.Sprintf("%s/pullzone/%d/setForceSSL", c.apiUrl, pullzoneId), bytes.NewReader(body))
		if err != nil {
			return PullzoneHostname{}, err
		}

		if resp.StatusCode != http.StatusNoContent {
			return PullzoneHostname{}, errors.New("forceSSL failed with " + resp.Status)
		}
	}

	return c.GetPullzoneHostname(pullzoneId, data.Id)
}

func (c *Client) GetPullzoneHostname(pullzoneId int64, id int64) (PullzoneHostname, error) {
	pullzone, err := c.GetPullzone(pullzoneId)
	if err != nil {
		return PullzoneHostname{}, err
	}

	for _, hostname := range pullzone.Hostnames {
		if hostname.Id == id {
			hostname.PullzoneId = pullzoneId
			return hostname, nil
		}
	}

	return PullzoneHostname{}, errors.New("Hostname not found")
}

func (c *Client) DeletePullzoneHostname(pullzoneId int64, hostname string) error {
	body, err := json.Marshal(map[string]interface{}{
		"Hostname": hostname,
	})

	if err != nil {
		return err
	}

	resp, err := c.doRequest(http.MethodDelete, fmt.Sprintf("%s/pullzone/%d/removeHostname", c.apiUrl, pullzoneId), bytes.NewReader(body))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return errors.New(resp.Status)
	}

	return nil
}
