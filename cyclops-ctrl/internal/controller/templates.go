package controller

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	cerbosSDK "github.com/cerbos/cerbos-sdk-go/cerbos"
	"github.com/gin-gonic/gin"

	"github.com/cyclops-ui/cyclops/cyclops-ctrl/internal/cerbos"
	"github.com/cyclops-ui/cyclops/cyclops-ctrl/internal/cluster/k8sclient"
	"github.com/cyclops-ui/cyclops/cyclops-ctrl/internal/mapper"
	"github.com/cyclops-ui/cyclops/cyclops-ctrl/internal/models/dto"
	"github.com/cyclops-ui/cyclops/cyclops-ctrl/internal/template"
)

type Templates struct {
	templatesRepo    *template.Repo
	kubernetesClient *k8sclient.KubernetesClient
	cerbos           *cerbos.CerbosSvc
}

func NewTemplatesController(
	templatesRepo *template.Repo,
	kubernetes *k8sclient.KubernetesClient,
	cerbosSvc *cerbos.CerbosSvc,
) *Templates {
	return &Templates{
		templatesRepo:    templatesRepo,
		kubernetesClient: kubernetes,
		cerbos:           cerbosSvc,
	}
}

// TODO kaj je ovo
func semantic(current string) string {
	if len(current) == 0 {
		return "v1"
	}

	version, _ := strconv.Atoi(current[1:])

	version++

	return fmt.Sprintf("v%d", version)
}

func (c *Templates) GetTemplate(ctx *gin.Context) {
	ctx.Header("Access-Control-Allow-Origin", "*")

	repo := ctx.Query("repo")
	path := ctx.Query("path")
	commit := ctx.Query("commit")

	if repo == "" {
		ctx.String(http.StatusBadRequest, "set repo field")
		return
	}

	t, err := c.templatesRepo.GetTemplate(repo, path, commit)
	if err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusBadRequest, dto.NewError("Error loading template", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, t)
}

func (c *Templates) GetTemplateInitialValues(ctx *gin.Context) {
	ctx.Header("Access-Control-Allow-Origin", "*")

	repo := ctx.Query("repo")
	path := ctx.Query("path")
	commit := ctx.Query("commit")

	if repo == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewError("Specify repo field", "Repo not specified"))
		return
	}

	initial, err := c.templatesRepo.GetTemplateInitialValues(repo, path, commit)
	if err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusBadRequest, dto.NewError("Error loading template initial values", err.Error()))
		return
	}

	ctx.Data(http.StatusOK, gin.MIMEJSON, initial)
}

func (c *Templates) ListTemplatesStore(ctx *gin.Context) {
	ctx.Header("Access-Control-Allow-Origin", "*")

	allowed := c.checkPermission(ctx, Resource.templatestore, "*", Action.list)
	if !allowed {
		errorMessage := fmt.Sprintf(
			"User does not have permission to perform '%s' action on %s",
			Action.list, Resource.templatestore,
		)
		ctx.JSON(http.StatusForbidden, dto.NewError("Permission Denied", errorMessage))
		return
	}

	store, err := c.kubernetesClient.ListTemplateStore()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Error fetching templates store", err.Error()))
		return
	}

	storeDTO := mapper.TemplateStoreListToDTO(store)

	ctx.JSON(http.StatusOK, storeDTO)
}

func (c *Templates) CreateTemplatesStore(ctx *gin.Context) {
	ctx.Header("Access-Control-Allow-Origin", "*")

	allowed := c.checkPermission(ctx, Resource.templatestore, "", Action.create)
	if !allowed {
		errorMessage := fmt.Sprintf(
			"User does not have permission to perform '%s' action on %s",
			Action.create, Resource.templatestore,
		)
		ctx.JSON(http.StatusForbidden, dto.NewError("Permission Denied", errorMessage))
		return
	}

	var templateStore *dto.TemplateStore
	if err := ctx.ShouldBind(&templateStore); err != nil {
		fmt.Println("error binding request", templateStore)
		ctx.JSON(http.StatusBadRequest, dto.NewError("Error binding request", err.Error()))
		return
	}

	templateStore.TemplateRef.URL = strings.Trim(templateStore.TemplateRef.URL, "/")
	templateStore.TemplateRef.Path = strings.Trim(templateStore.TemplateRef.Path, "/")
	templateStore.TemplateRef.Version = strings.Trim(templateStore.TemplateRef.Version, "/")

	if templateStore.TemplateRef.URL == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid template reference", "Template repo not set"))
		return
	}

	_, err := c.templatesRepo.GetTemplate(templateStore.TemplateRef.URL, templateStore.TemplateRef.Path, templateStore.TemplateRef.Version)
	if err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusBadRequest, dto.NewError("Error loading template", err.Error()))
		return
	}

	k8sTemplateStore := mapper.DTOToTemplateStore(*templateStore)

	if err := c.kubernetesClient.CreateTemplateStore(k8sTemplateStore); err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Error creating module", err.Error()))
		return
	}

	ctx.Status(http.StatusCreated)
}

func (c *Templates) EditTemplatesStore(ctx *gin.Context) {
	ctx.Header("Access-Control-Allow-Origin", "*")

	allowed := c.checkPermission(ctx, Resource.templatestore, ctx.Param("name"), Action.edit)
	if !allowed {
		errorMessage := fmt.Sprintf(
			"User does not have permission to perform '%s' action on %s",
			Action.edit, Resource.templatestore,
		)
		ctx.JSON(http.StatusForbidden, dto.NewError("Permission Denied", errorMessage))
		return
	}

	var templateStore *dto.TemplateStore
	if err := ctx.ShouldBind(&templateStore); err != nil {
		fmt.Println("error binding request", templateStore)
		ctx.JSON(http.StatusBadRequest, dto.NewError("Error binding request", err.Error()))
		return
	}

	templateStore.TemplateRef.URL = strings.Trim(templateStore.TemplateRef.URL, "/")
	templateStore.TemplateRef.Path = strings.Trim(templateStore.TemplateRef.Path, "/")
	templateStore.TemplateRef.Version = strings.Trim(templateStore.TemplateRef.Version, "/")

	if templateStore.TemplateRef.URL == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewError("Invalid template reference", "Template repo not set"))
		return
	}

	_, err := c.templatesRepo.GetTemplate(templateStore.TemplateRef.URL, templateStore.TemplateRef.Path, templateStore.TemplateRef.Version)
	if err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusBadRequest, dto.NewError("Error loading template", err.Error()))
		return
	}

	templateStore.Name = ctx.Param("name")

	k8sTemplateStore := mapper.DTOToTemplateStore(*templateStore)

	if err := c.kubernetesClient.UpdateTemplateStore(k8sTemplateStore); err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Error creating module", err.Error()))
		return
	}

	ctx.Status(http.StatusCreated)
}

func (c *Templates) DeleteTemplatesStore(ctx *gin.Context) {
	ctx.Header("Access-Control-Allow-Origin", "*")

	allowed := c.checkPermission(ctx, Resource.templatestore, ctx.Param("name"), Action.delete)
	if !allowed {
		errorMessage := fmt.Sprintf(
			"User does not have permission to perform '%s' action on %s",
			Action.delete, Resource.templatestore,
		)
		ctx.JSON(http.StatusForbidden, dto.NewError("Permission Denied", errorMessage))
		return
	}

	templateRefName := ctx.Param("name")

	if err := c.kubernetesClient.DeleteTemplateStore(templateRefName); err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewError("Error deleting module", err.Error()))
		return
	}

	ctx.Status(http.StatusOK)
}

func (c *Templates) checkPermission(ctx *gin.Context, kind, resourceName, action string) bool {
	if os.Getenv("CYCLOPS_AUTHORIZATION") == "disabled" {
		return true
	}
	resource := cerbosSDK.NewResource(kind, "new").
		WithAttr("name", resourceName).
		WithAttr("action", action)

	allowed, err := c.cerbos.IsAllowed(ctx.Request.Context(), resource, action)
	if err != nil {
		log.Println("Error checking permissions", err.Error())
		return false
	}
	return allowed
}
