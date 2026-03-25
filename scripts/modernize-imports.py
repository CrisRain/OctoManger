import os
import re

def process_file(filepath):
    with open(filepath, "r", encoding="utf-8") as f:
        content = f.read()
    
    # Remove import { ... } from "vue"
    content = re.sub(r"import\s+\{([^}]+)\}\s+from\s+[\"']vue[\"'];?\n", "", content)
    # Remove import { ... } from "vue-router"
    content = re.sub(r"import\s+\{([^}]+)\}\s+from\s+[\"']vue-router[\"'];?\n", "", content)
    # Remove component imports from @/components/index
    content = re.sub(r"import\s+\{([^}]+)\}\s+from\s+[\"']@/components(?:/index)?[\"'];?\n", "", content)
    # Remove empty lines left by deleted imports
    content = re.sub(r"\n\s*\n\s*\n", "\n\n", content)
    
    with open(filepath, "w", encoding="utf-8") as f:
        f.write(content)

for root, dirs, files in os.walk("apps/web/src"):
    for file in files:
        if file.endswith(".vue"):
            process_file(os.path.join(root, file))

print("All Vue files modernized successfully.")
