package project

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// Init creates a new Tracks application with the given module name and project name
func Init(moduleName, projectName string) error {
	// 1. Create project directory
	if err := os.MkdirAll(projectName, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// 2. Initialize Git repository
	if err := initGitRepository(projectName); err != nil {
		return err
	}

	// 3. Initialize Go module with the specified module name
	if err := initGoModule(projectName, moduleName); err != nil {
		return err
	}

	// 4. Create directory structure
	if err := createDirectoryStructure(projectName); err != nil {
		return err
	}

	// 5. Create initial files
	if err := createInitialFiles(projectName, moduleName); err != nil {
		return err
	}

	return nil
}

func initGitRepository(projectDir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Create .gitignore file
	gitignorePath := filepath.Join(projectDir, ".gitignore")
	gitignoreContent := `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
*.db

# Output directories
/bin/
/tmp/

# Go specific
/vendor/
/go.sum

# IDE files
.idea/
.vscode/
*.swp
*.swo
`
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore file: %w", err)
	}

	return nil
}

func initGoModule(projectDir, moduleName string) error {
	cmd := exec.Command("go", "mod", "init", moduleName)
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize go module: %w", err)
	}

	// Add tracks as a dependency
	cmd = exec.Command("go", "get", "github.com/tmeire/tracks")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add tracks dependency: %w", err)
	}

	return nil
}

func createDirectoryStructure(projectDir string) error {
	// Create standard directories
	dirs := []string{
		"controllers",
		"models",
		"views",
		"views/layouts",
		"views/default",
		"public",
		"public/css",
		"public/js",
		"public/images",
		"config",
		"data",
		"migrations/central",
	}

	for _, dir := range dirs {
		path := filepath.Join(projectDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		if err := os.WriteFile(filepath.Join(path, ".keep"), []byte(""), 0644); err != nil {
			return fmt.Errorf("failed to create .keep file in directory %s: %w", dir, err)
		}
	}

	return nil
}

func createInitialFiles(projectDir, moduleName string) error {
	// Create main.go
	if err := createMainFile(projectDir, moduleName); err != nil {
		return err
	}

	// Create default controller
	if err := createDefaultController(projectDir); err != nil {
		return err
	}

	// Create default view
	if err := createDefaultView(projectDir); err != nil {
		return err
	}

	// Create layout
	if err := createLayout(projectDir); err != nil {
		return err
	}

	// Create database configuration
	if err := createConfigFile(projectDir); err != nil {
		return err
	}

	// Create OpenTelemetry configuration
	//if err := createOtelConfig(projectDir); err != nil {
	//	return err
	//}

	// Create README.md
	if err := createReadme(projectDir); err != nil {
		return err
	}

	// Create CSS and JS files
	if err := createCssFile(projectDir); err != nil {
		return err
	}

	if err := createJsFile(projectDir); err != nil {
		return err
	}

	// Create Docker files
	if err := createDockerfile(projectDir); err != nil {
		return err
	}

	if err := createDockerCompose(projectDir); err != nil {
		return err
	}

	// Create observability configuration files
	if err := createOtelCollectorConfig(projectDir); err != nil {
		return err
	}

	if err := createPrometheusConfig(projectDir); err != nil {
		return err
	}

	return nil
}

func createMainFile(projectDir, moduleName string) error {
	appName := filepath.Base(projectDir)
	data := map[string]string{
		"PackageName": moduleName,
		"AppName":     appName,
	}

	return renderTemplate(
		"init/main.go.tmpl",
		filepath.Join(projectDir, "main.go"),
		data,
	)
}

func createDefaultController(projectDir string) error {
	return renderTemplate(
		"init/controllers/default.go.tmpl",
		filepath.Join(projectDir, "controllers", "default.go"),
		nil,
	)
}

func createDefaultView(projectDir string) error {
	return renderTemplate(
		"init/views/default/home.gohtml.tmpl",
		filepath.Join(projectDir, "views", "default", "home.gohtml"),
		nil,
	)
}

func createLayout(projectDir string) error {
	appName := filepath.Base(projectDir)
	data := map[string]string{
		"AppName": strings.ToUpper(appName[:1]) + appName[1:],
	}

	return renderTemplate(
		"init/views/layouts/application.gohtml.tmpl",
		filepath.Join(projectDir, "views", "layouts", "application.gohtml"),
		data,
	)
}

func createConfigFile(projectDir string) error {
	appName := filepath.Base(projectDir)
	data := map[string]string{
		"AppName": strings.ToLower(appName),
	}

	return renderTemplate(
		"init/config/config.json.tmpl",
		filepath.Join(projectDir, "config", "config.json"),
		data,
	)
}

func createOtelConfig(projectDir string) error {
	appName := filepath.Base(projectDir)
	data := map[string]string{
		"AppName": appName,
	}

	return renderTemplate(
		"init/config/otel.yml.tmpl",
		filepath.Join(projectDir, "config", "otel.yml"),
		data,
	)
}

func createCssFile(projectDir string) error {
	return renderTemplate(
		"init/public/css/application.css.tmpl",
		filepath.Join(projectDir, "public", "css", "application.css"),
		nil,
	)
}

func createJsFile(projectDir string) error {
	return renderTemplate(
		"init/public/js/application.js.tmpl",
		filepath.Join(projectDir, "public", "js", "application.js"),
		nil,
	)
}

func createReadme(projectDir string) error {
	appName := filepath.Base(projectDir)
	data := map[string]string{
		"AppName": strings.ToUpper(appName[:1]) + appName[1:],
	}

	return renderTemplate(
		"init/README.md.tmpl",
		filepath.Join(projectDir, "README.md"),
		data,
	)
}

func createDockerfile(projectDir string) error {
	appName := filepath.Base(projectDir)
	data := map[string]string{
		"AppName": appName,
	}

	return renderTemplate(
		"init/Dockerfile.tmpl",
		filepath.Join(projectDir, "Dockerfile"),
		data,
	)
}

func createDockerCompose(projectDir string) error {
	appName := filepath.Base(projectDir)
	data := map[string]string{
		"AppName": appName,
	}

	return renderTemplate(
		"init/docker-compose.yml.tmpl",
		filepath.Join(projectDir, "docker-compose.yml"),
		data,
	)
}

func createOtelCollectorConfig(projectDir string) error {
	return renderTemplate(
		"init/otel-collector.yaml.tmpl",
		filepath.Join(projectDir, "otel-collector.yaml"),
		nil,
	)
}

func createPrometheusConfig(projectDir string) error {
	appName := filepath.Base(projectDir)
	data := map[string]string{
		"AppName": appName,
	}

	return renderTemplate(
		"init/prometheus.yaml.tmpl",
		filepath.Join(projectDir, "prometheus.yaml"),
		data,
	)
}

func renderTemplate(templatePath, outputPath string, data map[string]string) error {
	// Ensure the directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	fileName := filepath.Base(templatePath)

	// Parse the template
	tmpl, err := template.New(fileName).Delims("<<", ">>").ParseFS(templateFS, "templates/"+templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templatePath, err)
	}

	// Create the output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputPath, err)
	}
	defer file.Close()

	fmt.Printf("%#v", tmpl.DefinedTemplates())

	// Execute the template
	if err := tmpl.ExecuteTemplate(file, fileName, data); err != nil {
		return fmt.Errorf("failed to execute template %q: %w", templatePath, err)
	}

	return nil
}
