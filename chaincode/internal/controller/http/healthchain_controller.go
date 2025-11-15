package http

import (
	"consentcc/internal/adapter/entity"
	"consentcc/internal/infra/client/fabric"
	"consentcc/internal/infra/server"
	"consentcc/internal/model"
	"consentcc/internal/service"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

type HealthChainController struct {
	consent service.HealthChainProvider
	fc      fabric.FabricClient
}

type IHealthChainController interface {
	Get(ctx echo.Context) error
	EvaluateFabric(ctx echo.Context) error
	CreateBundle(ctx echo.Context) error
	GrantConsent(ctx echo.Context) error
	RevokeConsent(ctx echo.Context) error
	//listAudits(ctx echo.Context) error
	ReadBundle(ctx echo.Context) error
	ReadConsent(ctx echo.Context) error
	GetBundlesByOwner(ctx echo.Context) error
}

func NewHTTPController(s server.EchoServer, consent service.HealthChainProvider, fc fabric.FabricClient) *HealthChainController {
	// Add more endpoints later (list consents, bundles, audits)

	controller := HealthChainController{consent: consent, fc: fc}
	return &controller

}

func (hc *HealthChainController) Get(ctx echo.Context) error {
	hc.fc.Contract().ContractName()
	return ctx.String(http.StatusOK, "HealthChainController is running")
}

func (c *HealthChainController) EvaluateFabric(ctx echo.Context) error {
	type EvaluateReq struct {
		Fcn  string   `json:"fcn"`
		Args []string `json:"args"`
	}

	req := new(EvaluateReq)
	if err := ctx.Bind(req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	// Calls a read-only chaincode function
	res, err := c.fc.Contract().EvaluateTransaction(req.Fcn, req.Args...)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "result": string(res)})
}

func (c *HealthChainController) listAudits(ctx echo.Context) error {
	//limit, offset := parsePage(ctx)
	//items, err := c.audit.List(ctx.Request().Context(), limit, offset)
	//if err != nil {
	//	return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	//}
	return ctx.JSON(http.StatusOK, nil)
}

func parsePage(c echo.Context) (int, int) {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func (c *HealthChainController) CreateBundle(ctx echo.Context) error {
	userId := ctx.Get("userId").(string)

	req := new(entity.CreateBundleReq)
	if err := ctx.Bind(req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	bundle, err := c.consent.CreateBundle(ctx.Request().Context(), req, userId)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "txResult": bundle})
}

func (c *HealthChainController) GrantConsent(ctx echo.Context) error {
	userId := ctx.Get("userId").(string)
	role := ctx.Get("role").(string)

	req := new(entity.GrantConsentReq)
	if err := ctx.Bind(req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	consent, err := c.consent.GrantConsent(ctx.Request().Context(), req, userId, role)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "txResult": consent})
}

func (c *HealthChainController) RevokeConsent(ctx echo.Context) error {
	req := new(entity.RevokeConsentReq)
	if err := ctx.Bind(req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	res, err := c.fc.Contract().SubmitTransaction(
		"RevokeConsent",
		req.BundleID, req.DoctorID,
	)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "txResult": string(res)})
}

func (c *HealthChainController) ReadBundle(ctx echo.Context) error {
	userId := ctx.Get("userId").(string)
	role := ctx.Get("role").(string)

	id := ctx.Param("id")
	res, err := c.fc.Contract().EvaluateTransaction("ReadBundle", id, userId, role)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	dataResponse := entity.Bundle{}
	_ = json.Unmarshal(res, &dataResponse)
	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "result": dataResponse})
}

func (c *HealthChainController) ReadConsent(ctx echo.Context) error {
	userId := ctx.Get("userId").(string)
	role := ctx.Get("role").(string)

	b := ctx.Param("bundleId")
	res, err := c.fc.Contract().EvaluateTransaction("ReadConsent", b, userId, role)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	dataResponse := entity.ConsentGrant{}
	_ = json.Unmarshal(res, &dataResponse)
	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "result": dataResponse})
}

func (c *HealthChainController) GetBundlesByOwner(ctx echo.Context) error {
	owner := ctx.Get("userId").(string)
	res, err := c.fc.Contract().EvaluateTransaction("GetBundlesByOwner", owner)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var bundles []entity.Bundle
	if err := json.Unmarshal(res, &bundles); err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return ctx.JSON(http.StatusOK, map[string]any{"ok": true, "result": bundles})
}

func (c *HealthChainController) ListMyConsents(ctx echo.Context) error {
	roleVal := ctx.Get("role")
	role, _ := roleVal.(string)

	if model.Role(role) != model.RolesConst.Patient {
		return ctx.JSON(http.StatusForbidden, map[string]string{"error": "only patient can list consents on their data"})
	}

	userVal := ctx.Get("userId")
	userId, _ := userVal.(string)
	if userId == "" {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{"error": "userId not in context"})
	}

	// Call Fabric: ListConsentsByOwner(ownerId=userId)
	res, err := c.fc.Contract().EvaluateTransaction("ListConsentsByOwner", userId, role)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var items []entity.ConsentWithBundle // mirror of model.ConsentWithBundle
	if err := json.Unmarshal(res, &items); err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "invalid chaincode response"})
	}

	return ctx.JSON(http.StatusOK, map[string]any{
		"ok":      true,
		"ownerId": userId,
		"data":    items,
	})
}
