const fs = require('fs');
const path = require('path');

// Simple YAML parser for config/branding.yaml structure
function parseBrandingConfig(yamlContent) {
  const config = {};
  let currentSection = null;

  const lines = yamlContent.split('\n');
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;

    if (!line.startsWith(' ') && line.endsWith(':')) {
      currentSection = line.replace(':', '').trim();
      config[currentSection] = {};
      continue;
    }

    if (currentSection && line.startsWith('  ')) {
      const colonIdx = trimmed.indexOf(':');
      if (colonIdx !== -1) {
        const key = trimmed.substring(0, colonIdx).trim();
        let val = trimmed.substring(colonIdx + 1).trim();
        // Strip inline comments if present (must have space before # or start with #)
        const commentMatch = val.match(/(^|\s+)#.*/);
        if (commentMatch) {
          val = val.substring(0, commentMatch.index).trim();
        }
        if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) {
          val = val.substring(1, val.length - 1);
        }
        config[currentSection][key] = val;
      }
    }
  }
  return config;
}

function main() {
  const rootDir = path.resolve(__dirname, '..');
  const configPath = path.join(rootDir, 'config', 'branding.yaml');

  if (!fs.existsSync(configPath)) {
    console.error(`Config file not found at ${configPath}`);
    process.exit(1);
  }

  const yamlContent = fs.readFileSync(configPath, 'utf8');
  const parsed = parseBrandingConfig(yamlContent);
  const branding = parsed.branding || {};

  const cssContent = `:root {
  --primary-color: ${branding.primary_color || '#2563eb'};
  --secondary-color: ${branding.secondary_color || '#ea580c'};
  --accent-color: ${branding.accent_color || '#3b82f6'};
  --background-color: ${branding.background_color || '#0f172a'};
  --card-color: ${branding.card_color || '#1e293b'};
  --text-color: ${branding.text_color || '#f8fafc'};
  --font-family: ${branding.font_family || 'Inter, system-ui, sans-serif'};
}
`;

  const targets = [
    path.join(rootDir, 'web', 'customer-portal', 'src', 'theme.css'),
    path.join(rootDir, 'web', 'employee-portal', 'src', 'theme.css')
  ];

  for (const targetPath of targets) {
    fs.mkdirSync(path.dirname(targetPath), { recursive: true });
    fs.writeFileSync(targetPath, cssContent, 'utf8');
    console.log(`Generated theme CSS: ${targetPath}`);
  }
}

main();
