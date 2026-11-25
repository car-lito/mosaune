package main

import "errors"

// Trusted adapters opt into repository dispatch with explicit package IDs.
// Remote catalogues cannot nominate executable handlers or grant trust.
type providerRepositoryManager interface {
	metadataPackageProvider
	repositoryPackageIDs() []string
}
type providerRepositoryBinding struct {
	PackageID  string `json:"packageId"`
	ProviderID string `json:"providerId"`
	Name       string `json:"name"`
}

func (c *catalog) providerRepositoryBindings() ([]providerRepositoryBinding, error) {
	bindings := []providerRepositoryBinding{}
	for _, provider := range c.providers {
		manager, ok := provider.(providerRepositoryManager)
		if !ok {
			continue
		}
		ids := manager.repositoryPackageIDs()
		if len(ids) > 128 {
			return nil, errors.New("Trop de paquets déclarés par un fournisseur.")
		}
		seen := map[string]bool{}
		for _, id := range ids {
			if !extensionID.MatchString(id) || len(id) > 100 || seen[id] {
				return nil, errors.New("Déclaration de paquets fournisseur invalide.")
			}
			seen[id] = true
			info := provider.Info()
			bindings = append(bindings, providerRepositoryBinding{PackageID: id, ProviderID: info.ID, Name: info.Name})
		}
	}
	return bindings, nil
}
func (c *catalog) repositoryProvider(id string) (string, metadataPackageProvider, error) {
	bindings, err := c.providerRepositoryBindings()
	if err != nil {
		return "", nil, err
	}
	selected := ""
	for _, binding := range bindings {
		if binding.PackageID != id {
			continue
		}
		if selected != "" {
			return "", nil, errors.New("Plusieurs fournisseurs prennent en charge ce paquet. Retirez l’ambiguïté avant de l’installer.")
		}
		selected = binding.ProviderID
	}
	if selected == "" {
		return "", nil, errors.New("L’adaptateur de ce fournisseur n’est pas chargé. Activez-le au lancement de Mosaune avant de télécharger ce paquet.")
	}
	manager, ok := c.provider(selected).(metadataPackageProvider)
	if !ok {
		return "", nil, errors.New("Gestionnaire fournisseur indisponible.")
	}
	return selected, manager, nil
}
