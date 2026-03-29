package http

import (
	"log"
	"net/http"

	"fcp/internal/adapter/entity"
	"fcp/internal/service"
	"github.com/labstack/echo/v4"
)

type FederationHandler struct {
	svc service.FederationService
}

func NewFederationHandler(svc service.FederationService) *FederationHandler {
	return &FederationHandler{svc: svc}
}

func (h *FederationHandler) Register(c echo.Context) error {
	var req service.RegisterReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if req.HospitalID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "hospital id is required"})
	}

	if len(req.ListHODID) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "field list hod is required"})
	}

	resp, err := h.svc.Register(c.Request().Context(), req)
	if err != nil {
		log.Default().Println(err.Error())
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"ok": true, "result": resp})
}

func (h *FederationHandler) Resolve(c echo.Context) error {
	hospitalID := c.Param("hospitalId")
	type resolveReq struct {
		InterfaceName entity.FederationInterfaceName `json:"interface_name"` // "PRIMARY" | "SECONDARY"
	}
	var req resolveReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	resp, err := h.svc.Resolve(c.Request().Context(), hospitalID, req.InterfaceName)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	log.Default().Println("Resolve Response", resp)
	return c.JSON(http.StatusOK, map[string]any{"ok": true, "result": resp})
}

type rotateKeyReq struct {
	PublicKey    string `json:"public_key"`
	KeyType      string `json:"key_type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// Hospital rotates its own key using issued client credentials (not admin)
func (h *FederationHandler) RotateKey(c echo.Context) error {
	hospitalID := c.Param("hospitalId")

	var req rotateKeyReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	ok, err := h.svc.VerifyClientSecret(c.Request().Context(), hospitalID, req.ClientID, req.ClientSecret)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid client credentials"})
	}

	resp, err := h.svc.RotateKey(c.Request().Context(), hospitalID, req.PublicKey, req.KeyType)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"ok": true, "result": resp})
}

type statusReq struct {
	Status string `json:"status"` // ACTIVE | DISABLED
}

func (h *FederationHandler) SetStatus(c echo.Context) error {
	hospitalID := c.Param("hospitalId")
	var req statusReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var st entity.HospitalStatus
	switch req.Status {
	case "ACTIVE":
		st = entity.HospitalActive
	case "DISABLED":
		st = entity.HospitalDisabled
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid status"})
	}

	resp, err := h.svc.SetStatus(c.Request().Context(), hospitalID, st)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"ok": true, "result": resp})
}
