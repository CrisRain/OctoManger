import fs from 'fs';
import path from 'path';

const pagesDir = path.join(process.cwd(), 'src/pages');

const routes = [
  'agents-list-page.tsx',
  'agent-create-page.tsx',
  'agent-detail-page.tsx',
  'agent-edit-page.tsx',

  'jobs-list-page.tsx',
  'job-create-page.tsx',
  'job-detail-page.tsx',
  'job-edit-page.tsx',
  'job-executions-list-page.tsx',
  'job-execution-detail-page.tsx',

  'account-types-list-page.tsx',
  'account-type-create-page.tsx',
  'account-type-edit-page.tsx',

  'accounts-list-page.tsx',
  'account-create-page.tsx',
  'account-edit-page.tsx',

  'triggers-list-page.tsx',
  'trigger-create-page.tsx',
  'trigger-edit-page.tsx',

  'email-accounts-list-page.tsx',
  'email-account-create-page.tsx',
  'email-account-edit-page.tsx',
  'email-account-preview-page.tsx',

  'plugins-list-page.tsx',
  'plugin-detail-page.tsx',

  'system-status-page.tsx',
  'system-config-page.tsx',
  'system-config-edit-page.tsx',
];

if (!fs.existsSync(pagesDir)) {
  fs.mkdirSync(pagesDir, { recursive: true });
}

for (const route of routes) {
  const filePath = path.join(pagesDir, route);
  if (!fs.existsSync(filePath)) {
    const componentName = route.replace(/\.tsx$/, '').split('-').map(p => p.charAt(0).toUpperCase() + p.slice(1)).join('');
    const content = `import { PageHeader } from "@/components/page-header";\n\nexport function ${componentName}() {\n  return (\n    <div className="space-y-4">\n      <PageHeader title="${componentName}" description="Placeholder" />\n      <div>Content for ${componentName}</div>\n    </div>\n  );\n}\n`;
    fs.writeFileSync(filePath, content);
  }
}
console.log("Scaffolded " + routes.length + " files.");
