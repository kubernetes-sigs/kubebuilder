// UncommentKustomizationBlocks uncomments specified blocks in kustomization.yaml
func UncommentKustomizationBlocks(path string, patterns []string) error {
    content, err := os.ReadFile(path)
    if err != nil {
        return fmt.Errorf("failed to read kustomization file: %w", err)
    }

    lines := strings.Split(string(content), "\n")
    modified := false

    for i, line := range lines {
        for _, pattern := range patterns {
            if strings.Contains(line, pattern) && strings.HasPrefix(strings.TrimSpace(line), "#") {
                lines[i] = strings.TrimPrefix(strings.TrimSpace(line), "#")
                modified = true
            }
        }
    }

    if modified {
        err = os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
        if err != nil {
            return fmt.Errorf("failed to write kustomization file: %w", err)
        }
    }

    return nil
} 
