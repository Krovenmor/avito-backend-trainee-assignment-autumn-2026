package http

import (
	"net/http"

	"avito-kitchen/services/kitchen/internal/httpx"
	"avito-kitchen/services/kitchen/internal/places/service"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

const (
	pathParamID         = "id"
	pathParamExternalID = "external_id"

	queryParamCity      = "city"
	queryParamType      = "type"
	queryParamMinRating = "min_rating"
	queryParamSort      = "sort"

	sortByRating = "rating"

	externalIDTag = "required,max=200"
)

type Handler struct {
	svc      service.Service
	validate *validator.Validate
	log      *zap.Logger
}

func NewHandler(svc service.Service, validate *validator.Validate, log *zap.Logger) *Handler {
	return &Handler{svc: svc, validate: validate, log: log}
}

func (h *Handler) Register(mux *http.ServeMux) {
	protected := func(f func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
		return h.requireAPIKey(h.requireOwnPlace(f))
	}

	mux.HandleFunc("POST /api/v1/b2b/places", h.registerPlace)
	mux.HandleFunc("PATCH /api/v1/b2b/places/{id}", protected(h.updatePlace))
	mux.HandleFunc("POST /api/v1/b2b/places/{id}/menu", protected(h.addMenuItems))
	mux.HandleFunc("PATCH /api/v1/b2b/places/{id}/menu/{external_id}", protected(h.updateMenuItem))
	mux.HandleFunc("DELETE /api/v1/b2b/places/{id}/menu/{external_id}", protected(h.deleteMenuItem))
	mux.HandleFunc("GET /api/v1/places", h.listPlaces)
	mux.HandleFunc("GET /api/v1/places/{id}", h.getPlace)
	mux.HandleFunc("GET /api/v1/places/{id}/menu", h.listMenu)
	mux.HandleFunc("GET /api/v1/products", h.searchProducts)
}

// @Summary Search dishes across places
// @Description Finds dishes by name across every active place, so a user can start from what they want to eat rather than from a place. Each item carries the place it belongs to.
// @Tags places
// @Produce json
// @Param search query string false "Dish name search (ILIKE)"
// @Param city query string false "City filter"
// @Param type query string false "Place type filter" Enums(restaurant, cafe, shop)
// @Param limit query int false "Page size, default 20, max 100"
// @Param from query int false "Offset, default 0"
// @Success 200 {array} productSearchItemResponse
// @Router /api/v1/products [get]
func (h *Handler) searchProducts(w http.ResponseWriter, r *http.Request) {
	limit, offset := httpx.ParsePagination(r)

	items, err := h.svc.SearchProducts(r.Context(), toSearchProductsParams(r, limit, offset))
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toProductSearchItemResponses(items))
}

// @Summary Register a place
// @Description Registers a new place. Returns the place_id and a one-time api_key.
// @Tags b2b/places
// @Accept json
// @Produce json
// @Param request body registerPlaceRequest true "Place registration data"
// @Success 201 {object} registerPlaceResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Router /api/v1/b2b/places [post]
func (h *Handler) registerPlace(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.DecodeAndValidate[registerPlaceRequest](h.validate, r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	out, err := h.svc.RegisterPlace(r.Context(), toRegisterPlaceInput(req))
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toRegisterPlaceResponse(out))
}

// @Summary Update a place
// @Description Partially updates a place's fields; only fields present in the body are changed.
// @Tags b2b/places
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path string true "Place ID"
// @Param request body updatePlaceRequest true "Fields to update"
// @Success 200
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /api/v1/b2b/places/{id} [patch]
func (h *Handler) updatePlace(w http.ResponseWriter, r *http.Request) {
	placeID := r.PathValue(pathParamID)

	req, err := httpx.DecodeAndValidate[updatePlaceRequest](h.validate, r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = h.svc.UpdatePlace(r.Context(), toUpdatePlaceInput(placeID, req))
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// @Summary Add menu items
// @Description Adds one or more products to a place's menu, always as an array. Returns the external_id to product_id mapping.
// @Tags b2b/places
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path string true "Place ID"
// @Param request body []menuItemRequest true "Menu items to add"
// @Success 201 {array} menuItemResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /api/v1/b2b/places/{id}/menu [post]
func (h *Handler) addMenuItems(w http.ResponseWriter, r *http.Request) {
	placeID := r.PathValue(pathParamID)

	req, err := httpx.DecodeAndValidateSlice[menuItemRequest](h.validate, r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	results, err := h.svc.AddMenuItems(r.Context(), placeID, toMenuItemInputs(req))
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toMenuItemResponses(results))
}

// @Summary Update a menu item
// @Description Overwrites name/description/price as given and applies stock_delta atomically to the current stock; reason is required when stock_delta is negative.
// @Tags b2b/places
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path string true "Place ID"
// @Param external_id path string true "Product external ID"
// @Param request body updateMenuItemRequest true "Fields to update"
// @Success 200
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /api/v1/b2b/places/{id}/menu/{external_id} [patch]
func (h *Handler) updateMenuItem(w http.ResponseWriter, r *http.Request) {
	placeID := r.PathValue(pathParamID)
	externalID, ok := httpx.StringPathParam(h.validate, w, r, pathParamExternalID, externalIDTag)
	if !ok {
		return
	}

	req, err := httpx.DecodeAndValidate[updateMenuItemRequest](h.validate, r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.StockDelta != nil && *req.StockDelta < 0 && req.Reason == "" {
		httpx.WriteError(w, http.StatusBadRequest, "reason is required when stock_delta is negative")
		return
	}

	err = h.svc.UpdateMenuItem(r.Context(), toUpdateMenuItemInput(placeID, externalID, req))
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// @Summary Delete a menu item
// @Description Soft-deletes a product from the menu; idempotent.
// @Tags b2b/places
// @Security ApiKeyAuth
// @Produce json
// @Param id path string true "Place ID"
// @Param external_id path string true "Product external ID"
// @Success 200
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /api/v1/b2b/places/{id}/menu/{external_id} [delete]
func (h *Handler) deleteMenuItem(w http.ResponseWriter, r *http.Request) {
	placeID := r.PathValue(pathParamID)
	externalID, ok := httpx.StringPathParam(h.validate, w, r, pathParamExternalID, externalIDTag)
	if !ok {
		return
	}

	err := h.svc.DeleteMenuItem(r.Context(), placeID, externalID)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// @Summary List places
// @Description Lists active places, optionally filtered by city, type and name search, with pagination.
// @Tags places
// @Produce json
// @Param city query string false "City filter"
// @Param type query string false "Place type filter" Enums(restaurant, cafe, shop)
// @Param search query string false "Name search (ILIKE)"
// @Param min_rating query number false "Only places rated at least this high; unrated places are excluded"
// @Param sort query string false "rating puts the best rated first; any other value keeps the default order" Enums(rating)
// @Param limit query int false "Page size, default 20, max 100"
// @Param from query int false "Offset, default 0"
// @Success 200 {array} placeListItemResponse
// @Router /api/v1/places [get]
func (h *Handler) listPlaces(w http.ResponseWriter, r *http.Request) {
	limit, offset := httpx.ParsePagination(r)
	places, err := h.svc.ListPlaces(r.Context(), toListPlacesParams(r, limit, offset))
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toPlaceListItemResponses(places))
}

// @Summary List a place's menu
// @Description Lists non-deleted products of a place with their stock_quantity, optionally filtered by name search, with pagination.
// @Tags places
// @Produce json
// @Param id path string true "Place ID"
// @Param search query string false "Name search (ILIKE)"
// @Param limit query int false "Page size, default 20, max 100"
// @Param from query int false "Offset, default 0"
// @Success 200 {array} productListItemResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /api/v1/places/{id}/menu [get]
func (h *Handler) listMenu(w http.ResponseWriter, r *http.Request) {
	placeID, ok := httpx.UUIDPathParam(h.validate, w, r, pathParamID)
	if !ok {
		return
	}

	limit, offset := httpx.ParsePagination(r)
	products, err := h.svc.ListProducts(r.Context(), toListProductsParams(r, placeID, limit, offset))
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toProductListItemResponses(products))
}

// @Summary Get one place
// @Description Everything a place page needs: address, working hours, whether it is open right now and the rating.
// @Tags places
// @Produce json
// @Param id path string true "Place ID"
// @Success 200 {object} placeListItemResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /api/v1/places/{id} [get]
func (h *Handler) getPlace(w http.ResponseWriter, r *http.Request) {
	placeID, ok := httpx.UUIDPathParam(h.validate, w, r, pathParamID)
	if !ok {
		return
	}

	place, err := h.svc.GetPlace(r.Context(), placeID)
	if err != nil {
		h.writeServiceError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toPlaceListItemResponse(*place))
}
