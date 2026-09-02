package service

import (
	"fmt"
	"strings"

	"whatsapp-gateway/model"
	"whatsapp-gateway/repository"
)

type TemplateService struct {
	repo *repository.TemplateRepo
}

func NewTemplateService(repo *repository.TemplateRepo) *TemplateService {
	return &TemplateService{repo: repo}
}

func (s *TemplateService) Resolve(name string, vars map[string]string) (string, *model.Template, error) {
	tmpl, err := s.repo.FindByName(name)
	if err != nil {
		return "", nil, err
	}
	if tmpl == nil {
		return "", nil, fmt.Errorf("template %q not found", name)
	}

	body := tmpl.Body
	for k, v := range vars {
		body = strings.ReplaceAll(body, "{{"+k+"}}", v)
	}
	return body, tmpl, nil
}

