package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type EdgeConfig struct {
	Slug   string `json:"slug"`
	ID     string `json:"id"`
	TeamID string `json:"ownerId"`
}

type CreateEdgeConfigRequest struct {
	Name   string `json:"slug"`
	TeamID string `json:"-"`
}

func (c *Client) CreateEdgeConfig(ctx context.Context, request CreateEdgeConfigRequest) (e EdgeConfig, err error) {
	url := fmt.Sprintf("%s/v1/edge-config", c.baseURL)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}
	payload := string(mustMarshal(request))
	tflog.Info(ctx, "creating edge config", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: http.MethodPost,
		url:    url,
		body:   payload,
	}, &e)
	return e, err
}

func (c *Client) GetEdgeConfig(ctx context.Context, id, teamID string) (e EdgeConfig, err error) {
	url := c.makeURL("/v1/edge-config/%s", id)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "reading edge config", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: http.MethodGet,
		url:    url,
	}, &e)
	return e, err
}

type UpdateEdgeConfigRequest struct {
	Slug   string `json:"slug"`
	TeamID string `json:"-"`
	ID     string `json:"-"`
}

func (c *Client) UpdateEdgeConfig(ctx context.Context, request UpdateEdgeConfigRequest) (e EdgeConfig, err error) {
	url := c.makeURL("/v1/edge-config/%s", request.ID)
	if c.teamID(request.TeamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(request.TeamID))
	}

	payload := string(mustMarshal(request))
	tflog.Trace(ctx, "updating edge config", map[string]interface{}{
		"url":     url,
		"payload": payload,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: http.MethodPut,
		url:    url,
		body:   payload,
	}, &e)
	return e, err
}

func (c *Client) DeleteEdgeConfig(ctx context.Context, id, teamID string) error {
	url := c.makeURL("/v1/edge-config/%s", id)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "deleting edge config", map[string]interface{}{
		"url": url,
	})

	return c.doRequest(clientRequest{
		ctx:    ctx,
		method: http.MethodDelete,
		url:    url,
	}, nil)
}

func (c *Client) ListEdgeConfigs(ctx context.Context, teamID string) (e []EdgeConfig, err error) {
	url := fmt.Sprintf("%s/v1/edge-config", c.baseURL)
	if c.teamID(teamID) != "" {
		url = fmt.Sprintf("%s?teamId=%s", url, c.teamID(teamID))
	}
	tflog.Info(ctx, "listing edge configs", map[string]interface{}{
		"url": url,
	})
	err = c.doRequest(clientRequest{
		ctx:    ctx,
		method: http.MethodGet,
		url:    url,
	}, &e)
	return e, err
}
