import { setupAuthUI } from "./auth.js";

// Knowledge Graph Visualization with D3.js

class KnowledgeGraph {
  constructor() {
    this.nodes = [];
    this.edges = [];
    this.types = [];
    this.stats = {};
    this.selectedNode = null;
    this.simulation = null;
    this.svg = null;
    this.g = null;
    this.zoom = null;
    
    this.typeColors = {
      user_preference: '#3498db',
      user_info: '#2ecc71',
      context: '#f39c12',
      fact: '#9b59b6'
    };

    this.edgeColors = {
      related_to: '#6b7280',
      implies: '#3b82f6',
      contradicts: '#ef4444',
      supports: '#22c55e'
    };

    this.filteredNodeTypes = new Set();
    this.filteredEdgeTypes = new Set();
    this.searchQuery = '';
  }

  async init() {
    this.setupEventListeners();
    this.setupGraph();
    await this.loadGraph();
  }

  setupEventListeners() {
    document.getElementById('search-input').addEventListener('input', (e) => {
      this.searchQuery = e.target.value.toLowerCase();
      this.updateGraphVisibility();
    });

    document.getElementById('refresh-graph').addEventListener('click', () => {
      this.loadGraph();
    });

    document.getElementById('reset-zoom').addEventListener('click', () => {
      this.resetZoom();
    });

    document.getElementById('fit-graph').addEventListener('click', () => {
      this.fitToScreen();
    });

    document.getElementById('close-detail').addEventListener('click', () => {
      this.closeDetailPanel();
    });
  }

  async loadGraph() {
    const loadingOverlay = document.getElementById('loading-overlay');
    const emptyState = document.getElementById('empty-state');
    
    loadingOverlay.style.display = 'flex';
    emptyState.style.display = 'none';

    try {
      const response = await fetch('/api/knowledge/graph');
      if (!response.ok) {
        throw new Error('Failed to load graph');
      }

      const data = await response.json();
      this.nodes = data.nodes || [];
      // Transform edges to d3 format (source/target instead of from_node_id/to_node_id)
      this.edges = (data.edges || []).map(edge => ({
        ...edge,
        source: edge.from_node_id,
        target: edge.to_node_id
      }));
      this.types = data.types || [];
      this.stats = data.stats || { total_nodes: 0, total_edges: 0, by_type: {} };

      this.updateTypeColors();
      this.updateStats();
      this.updateFilters();

      if (this.nodes.length === 0) {
        emptyState.style.display = 'flex';
        loadingOverlay.style.display = 'none';
        return;
      }

      if (this.link && this.node) {
        this.updateGraph();
      } else {
        this.renderGraph();
      }

      loadingOverlay.style.display = 'none';
    } catch (error) {
      console.error('Error loading graph:', error);
      loadingOverlay.innerHTML = '<p style="color: var(--error);">Failed to load graph</p>';
    }
  }

  updateTypeColors() {
    const nodeTypes = this.types.filter(t => t.category === 'node_type');
    nodeTypes.forEach(type => {
      if (!this.typeColors[type.name]) {
        this.typeColors[type.name] = this.generateColorFromString(type.name);
      }
    });

    const edgeTypes = this.types.filter(t => t.category === 'edge_type');
    edgeTypes.forEach(type => {
      if (!this.edgeColors[type.name]) {
        this.edgeColors[type.name] = this.generateColorFromString(type.name);
      }
    });
  }

  generateColorFromString(str) {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      hash = str.charCodeAt(i) + ((hash << 5) - hash);
    }
    
    const h = hash % 360;
    const s = 60 + (hash % 20);
    const l = 50 + (hash % 15);
    
    return `hsl(${h}, ${s}%, ${l}%)`;
  }

  updateStats() {
    document.getElementById('stat-nodes').textContent = this.stats.total_nodes || 0;
    document.getElementById('stat-edges').textContent = this.stats.total_edges || 0;

    const statsByType = document.getElementById('stats-by-type');
    statsByType.innerHTML = '';

    if (this.stats.by_type) {
      Object.entries(this.stats.by_type)
        .sort((a, b) => b[1] - a[1])
        .forEach(([type, count]) => {
          const color = this.typeColors[type] || '#6b7280';
          const div = document.createElement('div');
          div.className = 'type-stat';
          div.innerHTML = `
            <span class="type-stat-name">
              <span class="type-color-dot" style="background: ${color}"></span>
              ${type}
            </span>
            <span class="type-stat-count">${count}</span>
          `;
          statsByType.appendChild(div);
        });
    }
  }

  updateFilters() {
    const nodeTypeFilters = document.getElementById('node-type-filters');
    const edgeTypeFilters = document.getElementById('edge-type-filters');

    const nodeTypes = this.types.filter(t => t.category === 'node_type');
    const edgeTypes = this.types.filter(t => t.category === 'edge_type');

    const existingNodeTypes = new Set();
    const existingEdgeTypes = new Set();
    
    nodeTypeFilters.querySelectorAll('input:not([data-type="all"])').forEach(input => {
      existingNodeTypes.add(input.dataset.type);
    });
    edgeTypeFilters.querySelectorAll('input:not([data-edge-type="all"])').forEach(input => {
      existingEdgeTypes.add(input.dataset.edgeType);
    });

    nodeTypes.forEach(type => {
      if (!existingNodeTypes.has(type.name)) {
        const color = this.typeColors[type.name] || '#6b7280';
        const label = document.createElement('label');
        label.className = 'filter-item';
        label.innerHTML = `
          <input type="checkbox" checked data-type="${type.name}" />
          <span class="type-color-dot" style="background: ${color}"></span>
          <span>${type.name}</span>
        `;
        nodeTypeFilters.appendChild(label);

        label.querySelector('input').addEventListener('change', (e) => {
          this.handleNodeTypeFilter(e);
        });
      }
    });

    edgeTypes.forEach(type => {
      if (!existingEdgeTypes.has(type.name)) {
        const label = document.createElement('label');
        label.className = 'filter-item';
        label.innerHTML = `
          <input type="checkbox" checked data-edge-type="${type.name}" />
          <span>${type.name}</span>
        `;
        edgeTypeFilters.appendChild(label);

        label.querySelector('input').addEventListener('change', (e) => {
          this.handleEdgeTypeFilter(e);
        });
      }
    });

    const allNodeTypesCheckbox = nodeTypeFilters.querySelector('[data-type="all"]');
    if (allNodeTypesCheckbox && !allNodeTypesCheckbox.hasEventListener) {
      allNodeTypesCheckbox.hasEventListener = true;
      allNodeTypesCheckbox.addEventListener('change', (e) => {
        const checked = e.target.checked;
        nodeTypeFilters.querySelectorAll('input:not([data-type="all"])').forEach(input => {
          input.checked = checked;
        });
        this.filteredNodeTypes.clear();
        this.updateGraphVisibility();
      });
    }

    const allEdgeTypesCheckbox = edgeTypeFilters.querySelector('[data-edge-type="all"]');
    if (allEdgeTypesCheckbox && !allEdgeTypesCheckbox.hasEventListener) {
      allEdgeTypesCheckbox.hasEventListener = true;
      allEdgeTypesCheckbox.addEventListener('change', (e) => {
        const checked = e.target.checked;
        edgeTypeFilters.querySelectorAll('input:not([data-edge-type="all"])').forEach(input => {
          input.checked = checked;
        });
        this.filteredEdgeTypes.clear();
        this.updateGraphVisibility();
      });
    }
  }

  handleNodeTypeFilter(e) {
    const type = e.target.dataset.type;
    if (e.target.checked) {
      this.filteredNodeTypes.delete(type);
    } else {
      this.filteredNodeTypes.add(type);
    }
    this.updateGraphVisibility();
  }

  handleEdgeTypeFilter(e) {
    const type = e.target.dataset.edgeType;
    if (e.target.checked) {
      this.filteredEdgeTypes.delete(type);
    } else {
      this.filteredEdgeTypes.add(type);
    }
    this.updateGraphVisibility();
  }

  setupGraph() {
    const container = document.getElementById('graph-canvas');
    const svg = d3.select('#graph-svg');
    
    this.svg = svg;
    this.g = svg.append('g');

    const width = container.clientWidth;
    const height = container.clientHeight;

    this.zoom = d3.zoom()
      .scaleExtent([0.1, 4])
      .on('zoom', (event) => {
        this.g.attr('transform', event.transform);
      });

    svg.call(this.zoom);

    this.simulation = d3.forceSimulation()
      .force('link', d3.forceLink().id(d => d.id).distance(100))
      .force('charge', d3.forceManyBody().strength(-300))
      .force('center', d3.forceCenter(width / 2, height / 2))
      .force('collision', d3.forceCollide().radius(30));
  }

  renderGraph() {
    const width = document.getElementById('graph-canvas').clientWidth;
    const height = document.getElementById('graph-canvas').clientHeight;

    const linkGroup = this.g.append('g').attr('class', 'links');
    const nodeGroup = this.g.append('g').attr('class', 'nodes');

    const link = linkGroup
      .selectAll('line')
      .data(this.edges)
      .join('line')
      .attr('class', 'link')
      .attr('stroke', d => this.edgeColors[d.relation_type] || '#6b7280')
      .attr('marker-end', 'url(#arrowhead)');

    const defs = this.svg.append('defs');
    defs.append('marker')
      .attr('id', 'arrowhead')
      .attr('viewBox', '-0 -5 10 10')
      .attr('refX', 20)
      .attr('refY', 0)
      .attr('orient', 'auto')
      .attr('markerWidth', 6)
      .attr('markerHeight', 6)
      .append('path')
      .attr('d', 'M 0,-5 L 10,0 L 0,5')
      .attr('fill', '#6b7280');

    const node = nodeGroup
      .selectAll('g')
      .data(this.nodes)
      .join('g')
      .attr('class', 'node')
      .call(this.drag(this.simulation))
      .on('click', (event, d) => this.handleNodeClick(event, d));

    node.append('circle')
      .attr('r', 12)
      .attr('fill', d => this.typeColors[d.type] || '#6b7280');

    node.append('text')
      .attr('class', 'node-label')
      .attr('dy', 25)
      .text(d => this.truncateText(d.content, 20));

    this.simulation
      .nodes(this.nodes)
      .on('tick', () => {
        link
          .attr('x1', d => d.source.x)
          .attr('y1', d => d.source.y)
          .attr('x2', d => d.target.x)
          .attr('y2', d => d.target.y);

        node.attr('transform', d => `translate(${d.x},${d.y})`);
      });

    this.simulation.force('link').links(this.edges);
    this.simulation.alpha(1).restart();

    this.link = link;
    this.node = node;
  }

  updateGraph() {
    this.link
      .data(this.edges, d => d.id)
      .join('line')
      .attr('class', 'link')
      .attr('stroke', d => this.edgeColors[d.relation_type] || '#6b7280');

    this.node
      .data(this.nodes, d => d.id)
      .join(
        enter => {
          const g = enter.append('g')
            .attr('class', 'node')
            .call(this.drag(this.simulation))
            .on('click', (event, d) => this.handleNodeClick(event, d));

          g.append('circle')
            .attr('r', 12)
            .attr('fill', d => this.typeColors[d.type] || '#6b7280');

          g.append('text')
            .attr('class', 'node-label')
            .attr('dy', 25)
            .text(d => this.truncateText(d.content, 20));

          return g;
        }
      );

    this.simulation.nodes(this.nodes);
    this.simulation.force('link').links(this.edges);
    this.simulation.alpha(0.3).restart();
  }

  updateGraphVisibility() {
    if (!this.node || !this.link) return;

    this.node.classed('dimmed', d => {
      const typeFiltered = this.filteredNodeTypes.has(d.type);
      const searchFiltered = this.searchQuery && 
        !d.content.toLowerCase().includes(this.searchQuery) &&
        !d.type.toLowerCase().includes(this.searchQuery);
      
      return typeFiltered || searchFiltered;
    });

    this.link.classed('dimmed', d => {
      const edgeFiltered = this.filteredEdgeTypes.has(d.relation_type);
      const sourceNode = this.nodes.find(n => n.id === d.source.id || n.id === d.source);
      const targetNode = this.nodes.find(n => n.id === d.target.id || n.id === d.target);
      
      const sourceFiltered = sourceNode && (
        this.filteredNodeTypes.has(sourceNode.type) ||
        (this.searchQuery && 
         !sourceNode.content.toLowerCase().includes(this.searchQuery) &&
         !sourceNode.type.toLowerCase().includes(this.searchQuery))
      );
      
      const targetFiltered = targetNode && (
        this.filteredNodeTypes.has(targetNode.type) ||
        (this.searchQuery && 
         !targetNode.content.toLowerCase().includes(this.searchQuery) &&
         !targetNode.type.toLowerCase().includes(this.searchQuery))
      );

      return edgeFiltered || sourceFiltered || targetFiltered;
    });
  }

  drag(simulation) {
    function dragstarted(event) {
      if (!event.active) simulation.alphaTarget(0.3).restart();
      event.subject.fx = event.subject.x;
      event.subject.fy = event.subject.y;
    }

    function dragged(event) {
      event.subject.fx = event.x;
      event.subject.fy = event.y;
    }

    function dragended(event) {
      if (!event.active) simulation.alphaTarget(0);
      event.subject.fx = null;
      event.subject.fy = null;
    }

    return d3.drag()
      .on('start', dragstarted)
      .on('drag', dragged)
      .on('end', dragended);
  }

  async handleNodeClick(event, node) {
    event.stopPropagation();
    
    this.selectedNode = node;
    this.node.classed('selected', d => d.id === node.id);

    const detailPanel = document.getElementById('node-detail-panel');
    detailPanel.classList.add('visible');

    try {
      const response = await fetch(`/api/knowledge/node/${node.id}`);
      if (!response.ok) throw new Error('Failed to load node details');
      
      const data = await response.json();
      this.renderNodeDetail(data);
    } catch (error) {
      console.error('Error loading node details:', error);
      document.getElementById('detail-content').innerHTML = 
        '<p style="color: var(--error);">Failed to load node details</p>';
    }
  }

  renderNodeDetail(data) {
    const node = data.node;
    const neighbors = data.neighbors;
    const color = this.typeColors[node.type] || '#6b7280';

    let html = `
      <div class="detail-section">
        <div class="detail-label">Type</div>
        <div class="node-type-badge">
          <span class="type-color-dot" style="background: ${color}"></span>
          ${node.type}
        </div>
      </div>
    `;

    const distillFields = [
      ['Title', node.title],
      ['Description', node.description],
      ['Distillation reason', node.distillation_reason],
    ];
    distillFields.forEach(([label, val]) => {
      if (val != null && String(val).trim() !== '') {
        html += `
      <div class="detail-section">
        <div class="detail-label">${label}</div>
        <div class="detail-value">${this.escapeHtml(String(val))}</div>
      </div>`;
      }
    });

    html += `
      <div class="detail-section">
        <div class="detail-label">Content</div>
        <div class="detail-value">${this.escapeHtml(node.content)}</div>
      </div>
    `;

    if (node.search_text != null && String(node.search_text).trim() !== '') {
      html += `
      <div class="detail-section">
        <div class="detail-label">Search text</div>
        <pre class="detail-metadata">${this.escapeHtml(String(node.search_text))}</pre>
      </div>`;
    }

    html += `

      <div class="detail-section">
        <div class="detail-label">ID</div>
        <div class="detail-value" style="font-family: monospace; font-size: 12px;">${node.id}</div>
      </div>

      <div class="detail-section">
        <div class="detail-label">Created</div>
        <div class="detail-value">${this.formatDate(node.created_at)}</div>
      </div>
    `;

    if (node.metadata && Object.keys(node.metadata).length > 0) {
      html += `
        <div class="detail-section">
          <div class="detail-label">Metadata</div>
          <pre class="detail-metadata">${JSON.stringify(node.metadata, null, 2)}</pre>
        </div>
      `;
    }

    if (neighbors.nodes && neighbors.nodes.length > 1) {
      const neighborNodes = neighbors.nodes.filter(n => n.id !== node.id);
      
      if (neighborNodes.length > 0) {
        html += `
          <div class="detail-section">
            <div class="detail-label">Connected Nodes (${neighborNodes.length})</div>
            <div class="neighbor-list">
        `;

        neighborNodes.forEach(neighbor => {
          const edge = neighbors.edges.find(e => 
            (e.from_node_id === node.id && e.to_node_id === neighbor.id) ||
            (e.to_node_id === node.id && e.from_node_id === neighbor.id)
          );

          const relation = edge ? edge.relation_type : 'unknown';
          const direction = edge && edge.from_node_id === node.id ? '→' : '←';

          html += `
            <div class="neighbor-item" data-node-id="${neighbor.id}">
              <div class="neighbor-relation">${direction} ${relation}</div>
              <div class="neighbor-content">${this.escapeHtml(this.truncateText(neighbor.content, 100))}</div>
            </div>
          `;
        });

        html += `
            </div>
          </div>
        `;
      }
    }

    document.getElementById('detail-content').innerHTML = html;

    document.querySelectorAll('.neighbor-item').forEach(item => {
      item.addEventListener('click', async () => {
        const nodeId = item.dataset.nodeId;
        const node = this.nodes.find(n => n.id === nodeId);
        if (node) {
          this.handleNodeClick({ stopPropagation: () => {} }, node);
        }
      });
    });
  }

  closeDetailPanel() {
    document.getElementById('node-detail-panel').classList.remove('visible');
    this.node.classed('selected', false);
    this.selectedNode = null;
  }

  resetZoom() {
    this.svg.transition().duration(750).call(
      this.zoom.transform,
      d3.zoomIdentity
    );
  }

  fitToScreen() {
    if (this.nodes.length === 0) return;

    const bounds = this.g.node().getBBox();
    const parent = this.svg.node().parentElement;
    const fullWidth = parent.clientWidth;
    const fullHeight = parent.clientHeight;

    const width = bounds.width;
    const height = bounds.height;
    const midX = bounds.x + width / 2;
    const midY = bounds.y + height / 2;

    if (width === 0 || height === 0) return;

    const scale = 0.8 / Math.max(width / fullWidth, height / fullHeight);
    const translate = [
      fullWidth / 2 - scale * midX,
      fullHeight / 2 - scale * midY
    ];

    this.svg.transition().duration(750).call(
      this.zoom.transform,
      d3.zoomIdentity.translate(translate[0], translate[1]).scale(scale)
    );
  }

  truncateText(text, maxLength) {
    if (text.length <= maxLength) return text;
    return text.substring(0, maxLength) + '...';
  }

  escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  formatDate(dateStr) {
    try {
      const date = new Date(dateStr);
      return date.toLocaleString();
    } catch (e) {
      return dateStr;
    }
  }
}

const graph = new KnowledgeGraph();
(async () => {
  await setupAuthUI();
  await graph.init();
})();
