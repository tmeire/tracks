package project

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates
var templateFS embed.FS

type Project struct {
	rootDir string
}

func Load() (*Project, error) {
	rootDir, err := rootDir()
	if err != nil {
		return nil, err
	}

	return &Project{rootDir: rootDir}, nil
}

func (p *Project) Initialize() error {
	// Add tracks as a dependency in the go.mod file
	fmt.Println("Adding tracks as a dependency...")
	if err := p.addTracksDependency(); err != nil {
		return err
	}

	// Create base folders
	fmt.Println("Creating base folders...")
	if err := p.createBaseFolders(); err != nil {
		return err
	}

	applicationName := filepath.Base(p.rootDir)

	// Create landing page
	fmt.Println("Creating index page with layout and style...")
	if err := p.createLandingPage(applicationName); err != nil {
		return err
	}

	// Add router setup to main.go
	fmt.Println("Adding router setup to main.go...")
	return p.addRouterSetup()
}

// addTracksDependency adds tracks as a dependency in the go.mod file
func (p *Project) addTracksDependency() error {
	cmd := exec.Command("go", "get", "github.com/tmeire/tracks")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// createBaseFolders creates the base folders for the application
func (p *Project) createBaseFolders() error {
	folders := []string{"controllers", "models", "views/layouts", "views/default", "public/css"}
	for _, folder := range folders {
		if err := os.MkdirAll(folder, 0755); err != nil {
			return fmt.Errorf("Error creating %s directory: %w\n", folder, err)
		}
	}
	return nil
}

// createLandingPage creates a basic index page in the views directory, an application layout file and a css file
func (p *Project) createLandingPage(applicationName string) error {
	err := p.createFile("views/layouts/application.gohtml", "templates/application.gohtml.tmpl", nil)
	if err != nil {
		return err
	}

	err = p.createFile("views/default/index.gohtml", "templates/index.gohtml.tmpl", map[string]string{
		"ApplicationName": applicationName,
	})
	if err != nil {
		return err
	}

	err = p.createFile("public/css/application.css", "templates/application.css.tmpl", nil)
	if err != nil {
		return err
	}
	return nil
}

// addRouterSetup adds router setup to main.go
func (p *Project) addRouterSetup() error {
	// Check if main.go exists
	if _, err := os.Stat("main.go"); os.IsNotExist(err) {
		// Create main.go if it doesn't exist
		return p.createFile("main.go", "templates/main.go.tmpl", nil)
	}

	// Read the content of main.go
	content, err := os.ReadFile("main.go")
	if err != nil {
		return fmt.Errorf("error reading main.go: %w\n", err)
	}

	// Check if tracks is already imported
	contentStr := string(content)
	if !strings.Contains(contentStr, "github.com/tmeire/tracks") {
		// Add tracks import
		contentStr = strings.Replace(contentStr, "import (", "import (\n\t\"github.com/tmeire/tracks\"", 1)
	}

	// Check if router setup already exists
	if !strings.Contains(contentStr, "tracks.NewRouter()") {
		// Find the main function
		mainFuncIndex := strings.Index(contentStr, "func main() {")
		if mainFuncIndex == -1 {
			return fmt.Errorf("could not find main function in main.go")
		}

		// Find the opening brace of the main function
		openBraceIndex := strings.Index(contentStr[mainFuncIndex:], "{") + mainFuncIndex
		if openBraceIndex == -1 {
			return fmt.Errorf("could not find opening brace of main function in main.go")
		}

		// Insert router setup after the opening brace
		routerSetup := "\n\trouter := tracks.NewRouter()\n\n\t// Register the index page\n\trouter.Page(\"/\", \"index\")\n\n\trouter.Run()\n"
		contentStr = contentStr[:openBraceIndex+1] + routerSetup + contentStr[openBraceIndex+1:]
	}

	// Write the updated content back to main.go
	if err := os.WriteFile("main.go", []byte(contentStr), 0644); err != nil {
		return fmt.Errorf("error writing to main.go: %w\n", err)
	}
	return nil
}

func (p *Project) Assets() string {
	return filepath.Join(p.rootDir, "public")
}

func (p *Project) controllers() string {
	return filepath.Join(p.rootDir, "controllers")
}

func (p *Project) Views() string {
	return filepath.Join(p.rootDir, "views")
}

func (p *Project) models() string {
	return filepath.Join(p.rootDir, "models")
}

// AddResource creates a resource with controller, model, and views
func (p *Project) AddResource(resourceName string) error {
	// Convert resource name to various forms
	resourceNameCamel := actionNameToCamelCase(resourceName)
	resourceNamePlural := resourceName   // For simplicity, we're not handling pluralization
	resourceNameSingular := resourceName // For simplicity, we're not handling singularization

	// Create the model file
	err := p.createModelFile(resourceNameCamel, resourceNameSingular, resourceNamePlural)
	if err != nil {
		return err
	}

	// Create the controller file
	err = p.createResourceController(resourceNameCamel, resourceNameSingular, resourceNamePlural)
	if err != nil {
		return err
	}

	// Create view files
	err = p.createResourceViews(resourceNameCamel, resourceNameSingular, resourceNamePlural)
	if err != nil {
		return err
	}

	// Update main.go to register the resource
	err = p.registerResource(resourceNameCamel)
	if err != nil {
		return err
	}

	return nil
}

func (p *Project) createFile(filePath string, templateName string, data map[string]string) error {
	base := filepath.Dir(filePath)
	// Create the controllers directory if it doesn't exist
	err := os.MkdirAll(base, 0755)
	if err != nil {
		return err
	}

	// Check if the file exists
	_, err = os.Stat(filePath)
	if os.IsNotExist(err) {
		// File doesn't exist, create it from the template
		tmpl, err := template.ParseFS(templateFS, templateName)
		if err != nil {
			return err
		}

		// Create the file
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer func() error {
			return file.Close()
		}()

		// Execute the template
		err = tmpl.Execute(file, data)
		if err != nil {
			return err
		}

		fmt.Printf("Created file: %s\n", filePath)
	} else if err == nil {
		fmt.Printf("File %s already exists\n", filePath)
	} else {
		return err
	}

	return nil
}

// createModelFile creates a model file for the resource
func (p *Project) createModelFile(modelName, resourceSingular, resourcePath string) error {
	return p.createFile(
		filepath.Join(p.models(), strings.ToLower(modelName)+".go"),
		"templates/resource/model.go.tmpl",
		map[string]string{
			"ModelName":        modelName,
			"ResourceSingular": resourceSingular,
		})
}

// createResourceController creates a controller file for the resource
func (p *Project) createResourceController(resourceName, resourceSingular, resourcePath string) error {
	return p.createFile(
		filepath.Join(p.controllers(), strings.ToLower(resourcePath)+".go"),
		"templates/resource/controller.go.tmpl",
		map[string]string{
			"ResourceName":     resourceName,
			"ModelName":        resourceName,
			"ResourcePath":     resourcePath,
			"ResourceSingular": resourceSingular,
		})
}

func (p *Project) insertControllerRegistration(registrationLine string) error {
	// Read the main.go file
	mainFilePath := filepath.Join(p.rootDir, "main.go")

	content, err := os.ReadFile(mainFilePath)
	if err != nil {
		return err
	}

	// Check if the import for the controllers package is present
	if !strings.Contains(string(content), `/controllers"`) {
		return fmt.Errorf("main.go doesn't import the controllers package")
	}

	// Find the position to insert the new resource registration
	lines := strings.Split(string(content), "\n")
	insertIndex := -1
	for i, line := range lines {
		if strings.Contains(line, "tracks.New()") {
			insertIndex = i
		}
	}

	if insertIndex == -1 {
		// fallback to inserting at the first line of the main function
		for i, line := range lines {
			if strings.Contains(line, "func main() {") {
				insertIndex = i + 1
			}
		}
		if insertIndex == -1 {
			return fmt.Errorf("couldn't find a suitable position to insert the resource registration")
		}

		lines = append(lines[:insertIndex], append([]string{"t := tracks.New()."}, lines[insertIndex:]...)...)
		insertIndex++
	}

	// If the tracks.New() line does not end with a period, add one to make sure the syntax is valid
	// If it has, make sure the inserted line also ends with a period
	if !strings.HasSuffix(strings.TrimSpace(lines[insertIndex-1]), ".") {
		lines[insertIndex-1] += "."
	} else {
		if !strings.HasSuffix(strings.TrimSpace(registrationLine), ".") {
			registrationLine += "."
		}
	}

	// Insert the registration line
	newLines := append(lines[:insertIndex], append([]string{registrationLine}, lines[insertIndex:]...)...)
	newContent := strings.Join(newLines, "\n")

	// Write the updated content back to main.go
	err = os.WriteFile(mainFilePath, []byte(newContent), 0644)
	if err != nil {
		return err
	}
	return nil
}

// RegisterAction updates main.go to register the action
func (p *Project) registerAction(method, path, controllerName, actionName string) error {
	registrationLine := fmt.Sprintf("\t%s(\"%s\", \"%s\", \"%s\", controllers.%s)",
		methodToFuncName(method), path, controllerName, actionName, actionNameToCamelCase(actionName))

	fmt.Printf("Registering action %s#%s at path %s %s\n", controllerName, actionName, method, path)
	return p.insertControllerRegistration(registrationLine)
}

// registerResource updates main.go to register the resource
func (p *Project) registerResource(resourceName string) error {
	fmt.Printf("Registering resource %s\n", resourceName)
	return p.insertControllerRegistration(fmt.Sprintf("\t\tResource(&controllers.%s{})", resourceName))
}

// createResourceViews creates view files for the resource
func (p *Project) createResourceViews(resourceName, resourceSingular, resourcePath string) error {
	viewDir := filepath.Join(p.Views(), resourcePath)

	// Create each view file
	views := []string{"index", "new", "edit", "show"}
	for _, view := range views {
		err := p.createFile(
			filepath.Join(viewDir, view+".gohtml"),
			"templates/resource/"+view+".gohtml.tmpl",
			map[string]string{
				"ResourceName":     resourceName,
				"ResourcePath":     resourcePath,
				"ResourceSingular": resourceSingular,
			})
		if err != nil {
			return err
		}
	}

	return nil
}

// AddController creates a controller file if it doesn't exist
// If the file exists, it adds the action function to it.
func (p *Project) AddController(controllerName, actionName string) error {
	// Create the controllers directory if it doesn't exist
	err := os.MkdirAll(p.controllers(), 0755)
	if err != nil {
		return err
	}

	// Construct the file path
	filePath := filepath.Join(p.controllers(), controllerName+".go")

	// Check if the file exists
	_, err = os.Stat(filePath)
	if os.IsNotExist(err) {
		// File doesn't exist, create it from the template
		return p.createFile(
			filePath,
			"templates/controller/controller.go.tmpl",
			map[string]string{
				"ControllerName": controllerName,
				"ActionName":     actionNameToCamelCase(actionName),
			})
	}
	if err != nil {
		return err
	}

	// File exists, add the action function to it
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Check if the action function already exists
	if strings.Contains(string(content), fmt.Sprintf("func %s", actionNameToCamelCase(actionName))) {
		fmt.Printf("Action %s already exists in controller %s\n", actionName, controllerName)
		return nil
	}

	// Add the action function to the file
	newContent := string(content)
	// Find the last closing brace
	lastBraceIndex := strings.LastIndex(newContent, "}")
	if lastBraceIndex == -1 {
		return fmt.Errorf("invalid controller file format")
	}

	// Insert the action function before the last closing brace
	actionFunc := fmt.Sprintf(`

// %s is an action for %s
func %s(r *http.Request) (any, error) {
	return nil, nil
}
`, actionNameToCamelCase(actionName), controllerName, actionNameToCamelCase(actionName))

	newContent = newContent[:lastBraceIndex] + actionFunc + newContent[lastBraceIndex:]

	err = os.WriteFile(filePath, []byte(newContent), 0644)
	if err != nil {
		return err
	}

	fmt.Printf("Added action %s to controller %s\n", actionName, controllerName)
	return nil
}

// AddView creates a view template file ./views/controllerName/actionName.gohtml
func (p *Project) AddView(controllerName, actionName string) error {
	filePath := filepath.Join(p.Views(), controllerName, actionName+".gohtml")

	return p.createFile(
		filePath,
		"templates/resource/view.gohtml.tmpl",
		map[string]string{
			"ControllerName": controllerName,
			"ActionName":     actionName,
			"FilePath":       filePath,
		})
}

func (p *Project) AddPage(method string, path string) {
	// Extract controller and action names from the path
	controllerName, actionName := extractNames(path)

	fmt.Printf("Generating controller %s with action %s for method %s and path %s\n",
		controllerName, actionName, method, path)

	// Create the controller file if it doesn't exist
	err := p.AddController(controllerName, actionName)
	if err != nil {
		fmt.Printf("Error creating controller file: %v\n", err)
		os.Exit(1)
	}

	// Create the view file
	err = p.AddView(controllerName, actionName)
	if err != nil {
		fmt.Printf("Error creating view file: %v\n", err)
		os.Exit(1)
	}

	// Update main.go to register the action
	err = p.registerAction(method, path, controllerName, actionName)
	if err != nil {
		fmt.Printf("Error updating main.go: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Controller generated successfully!")
}

// rootDir attempts to find the root directory of the project
// by looking for the go.mod file or other project-specific files
func rootDir() (string, error) {
	// Start from the current directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Try to find the go.mod file by walking up the directory tree
	for {
		// Check if go.mod exists in the current directory
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		// Check if we've reached the root directory
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// If we couldn't find the go.mod file, return an error
	return "", fmt.Errorf("could not find project root (no go.mod file found)")
}

// actionNameToCamelCase converts an action name to CamelCase
func actionNameToCamelCase(name string) string {
	// Split the name by underscores or hyphens
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-'
	})

	// Capitalize the first letter of each part
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}

	// Join the parts
	return strings.Join(parts, "")
}

// methodToFuncName converts an HTTP method to the corresponding router function name
func methodToFuncName(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return "GetFunc"
	case "POST":
		return "PostFunc"
	case "PUT":
		return "PutFunc"
	case "PATCH":
		return "PatchFunc"
	case "DELETE":
		return "DeleteFunc"
	default:
		return "GetFunc" // Default to GetFunc
	}
}

// extractNames extracts the controller and action names from the path
// The controller and action names are the last two parts of the path
// If there's only one part, that part is the action name and the controller name is "default"
func extractNames(path string) (string, string) {
	// Remove leading slash if present
	path = strings.TrimPrefix(path, "/")

	// Split the path by slashes
	parts := strings.Split(path, "/")

	// If there's only one part, use "default" as the controller name
	if len(parts) == 1 {
		return "default", parts[0]
	}

	// Otherwise, use the last two parts as controller and action names
	return parts[len(parts)-2], parts[len(parts)-1]
}
