export class FileSystemManager {
    constructor() {
        this.modal = document.getElementById('fs-modal');
        this.toggleBtn = document.getElementById('workspace-toggle');
        this.closeBtn = document.getElementById('fs-close');
        this.treeContainer = document.getElementById('fs-tree');
        this.fileContent = document.getElementById('fs-file-content');
        this.filePre = document.getElementById('fs-file-pre');
        this.placeholder = document.getElementById('fs-viewer-placeholder');

        this.toggleBtn.addEventListener('click', () => this.open());
        this.closeBtn.addEventListener('click', () => this.close());

        // Close on click outside
        this.modal.addEventListener('click', (e) => {
            if (e.target === this.modal) this.close();
        });
    }

    async open() {
        this.modal.classList.remove('hidden');
        await this.loadDirectory('.');
    }

    close() {
        this.modal.classList.add('hidden');
    }

    async loadDirectory(path) {
        try {
            const res = await fetch(`/api/fs/list?path=${encodeURIComponent(path)}`);
            if (!res.ok) throw new Error('Failed to load directory');
            const data = await res.json();

            this.treeContainer.innerHTML = '';

            // Add "up" directory if not at root
            if (path !== '.' && path !== '') {
                const parentPath = path.split('/').slice(0, -1).join('/') || '.';
                const upIter = this.createNodeElement({ name: '..', path: parentPath, is_dir: true }, true);
                this.treeContainer.appendChild(upIter);
            }

            // Sort: directories first, then files
            const nodes = (data.nodes || []).sort((a, b) => {
                if (a.is_dir === b.is_dir) return a.name.localeCompare(b.name);
                return a.is_dir ? -1 : 1;
            });

            nodes.forEach(node => {
                const el = this.createNodeElement(node, false);
                this.treeContainer.appendChild(el);
            });
        } catch (err) {
            console.error(err);
            this.treeContainer.innerHTML = `<div class="error" style="color: var(--error)">Error loading directory</div>`;
        }
    }

    createNodeElement(node, isUpDir) {
        const el = document.createElement('div');
        el.className = 'fs-tree-node';

        const icon = document.createElement('span');
        icon.className = 'fs-tree-icon';
        icon.textContent = node.is_dir ? '📁' : '📄';
        if (isUpDir) icon.textContent = '↵';

        const name = document.createElement('span');
        name.className = 'fs-tree-name';
        name.textContent = node.name;

        el.appendChild(icon);
        el.appendChild(name);

        el.addEventListener('click', () => {
            // Remove active class from all
            document.querySelectorAll('.fs-tree-node').forEach(n => n.classList.remove('active'));
            el.classList.add('active');

            if (node.is_dir) {
                this.loadDirectory(node.path);
            } else {
                this.loadFile(node.path);
            }
        });

        return el;
    }

    async loadFile(path) {
        this.placeholder.classList.add('hidden');
        this.filePre.classList.remove('hidden');
        this.fileContent.textContent = 'Loading...';

        try {
            const res = await fetch(`/api/fs/read?path=${encodeURIComponent(path)}`);
            if (!res.ok) throw new Error('Failed to load file');

            const contentType = res.headers.get('content-type');
            if (contentType && contentType.startsWith('image/')) {
                this.fileContent.innerHTML = `<img src="/api/fs/read?path=${encodeURIComponent(path)}" style="max-width: 100%; border-radius: 4px;" />`;
            } else {
                const text = await res.text();
                this.fileContent.textContent = text;
            }
        } catch (err) {
            console.error(err);
            this.fileContent.textContent = 'Error loading file content.';
        }
    }
}
