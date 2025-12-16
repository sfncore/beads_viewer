package analysis

import (
	"math/rand"
	"sort"
	"time"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/network"
	"gonum.org/v1/gonum/graph/simple"
)

// BetweennessMode specifies how betweenness centrality should be computed.
type BetweennessMode string

const (
	// BetweennessExact computes exact betweenness centrality using Brandes' algorithm.
	// Complexity: O(V*E) - fast for small graphs, slow for large graphs.
	BetweennessExact BetweennessMode = "exact"

	// BetweennessApproximate uses sampling-based approximation.
	// Complexity: O(k*E) where k is the sample size - much faster for large graphs.
	// Error: O(1/sqrt(k)) - with k=100, ~10% error in ranking.
	BetweennessApproximate BetweennessMode = "approximate"

	// BetweennessSkip disables betweenness computation entirely.
	BetweennessSkip BetweennessMode = "skip"
)

// BetweennessResult contains the result of betweenness computation.
type BetweennessResult struct {
	// Scores maps node IDs to their betweenness centrality scores
	Scores map[int64]float64

	// Mode indicates how the result was computed
	Mode BetweennessMode

	// SampleSize is the number of pivot nodes used (only for approximate mode)
	SampleSize int

	// TotalNodes is the total number of nodes in the graph
	TotalNodes int

	// Elapsed is the time taken to compute
	Elapsed time.Duration

	// TimedOut indicates if computation was interrupted by timeout
	TimedOut bool
}

// ApproxBetweenness computes approximate betweenness centrality using sampling.
//
// Instead of computing shortest paths from ALL nodes (O(V*E)), we sample k pivot
// nodes and extrapolate. This is Brandes' approximation algorithm.
//
// Error bounds: With k samples, approximation error is O(1/sqrt(k)):
//   - k=50: ~14% error
//   - k=100: ~10% error
//   - k=200: ~7% error
//
// For ranking purposes (which node is most central), this is usually sufficient.
//
// References:
//   - "A Faster Algorithm for Betweenness Centrality" (Brandes, 2001)
//   - "Approximating Betweenness Centrality" (Bader et al., 2007)
func ApproxBetweenness(g *simple.DirectedGraph, sampleSize int, seed int64) BetweennessResult {
	start := time.Now()
	nodes := graph.NodesOf(g.Nodes())
	n := len(nodes)
	// Ensure deterministic ordering before sampling; gonum's Nodes may be map-backed.
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID() < nodes[j].ID() })

	result := BetweennessResult{
		Scores:     make(map[int64]float64),
		Mode:       BetweennessApproximate,
		SampleSize: sampleSize,
		TotalNodes: n,
	}

	if n == 0 {
		result.Elapsed = time.Since(start)
		return result
	}

	// For small graphs or when sample size >= node count, use exact algorithm
	if sampleSize >= n {
		exact := network.Betweenness(g)
		result.Scores = exact
		result.Mode = BetweennessExact
		result.SampleSize = n
		result.Elapsed = time.Since(start)
		return result
	}

	// Sample k random pivot nodes
	pivots := sampleNodes(nodes, sampleSize, seed)

	// Compute partial betweenness from sampled pivots only
	partialBC := make(map[int64]float64)
	for _, pivot := range pivots {
		singleSourceBetweenness(g, pivot, partialBC)
	}

	// Scale up: BC_approx = BC_partial * (n / k)
	// This extrapolates from the sample to the full graph
	scale := float64(n) / float64(sampleSize)
	for id := range partialBC {
		partialBC[id] *= scale
	}

	result.Scores = partialBC
	result.Elapsed = time.Since(start)
	return result
}

// sampleNodes returns a random sample of k nodes from the input slice.
// Uses Fisher-Yates shuffle for unbiased sampling.
func sampleNodes(nodes []graph.Node, k int, seed int64) []graph.Node {
	if k >= len(nodes) {
		return nodes
	}

	// Create a copy to avoid modifying the original
	shuffled := make([]graph.Node, len(nodes))
	copy(shuffled, nodes)

	// Fisher-Yates shuffle for first k elements
	rng := rand.New(rand.NewSource(seed))
	for i := 0; i < k; i++ {
		j := i + rng.Intn(len(shuffled)-i)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled[:k]
}

// singleSourceBetweenness computes the betweenness contribution from a single source node.
// This is the core of Brandes' algorithm, run once per pivot.
//
// The algorithm performs BFS from the source and accumulates dependency scores
// in a reverse topological order traversal.
func singleSourceBetweenness(g *simple.DirectedGraph, source graph.Node, bc map[int64]float64) {
	sourceID := source.ID()
	nodes := graph.NodesOf(g.Nodes())

	// Data structures for Brandes' algorithm
	sigma := make(map[int64]float64) // Number of shortest paths through node
	dist := make(map[int64]int)      // Distance from source
	delta := make(map[int64]float64) // Dependency of source on node
	pred := make(map[int64][]int64)  // Predecessors on shortest paths

	// Initialization
	for _, n := range nodes {
		nid := n.ID()
		sigma[nid] = 0
		dist[nid] = -1 // -1 means unreachable
		delta[nid] = 0
		pred[nid] = make([]int64, 0)
	}

	sigma[sourceID] = 1
	dist[sourceID] = 0

	// Queue for BFS
	queue := []int64{sourceID}

	// Stack for reverse traversal (accumulation)
	var stack []int64

	// BFS phase
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		stack = append(stack, v)

		to := g.From(v) // Outgoing edges
		for to.Next() {
			w := to.Node().ID()

			// Path discovery
			if dist[w] < 0 {
				dist[w] = dist[v] + 1
				queue = append(queue, w)
			}

			// Path counting
			if dist[w] == dist[v]+1 {
				sigma[w] += sigma[v]
				pred[w] = append(pred[w], v)
			}
		}
	}

	// Accumulation phase
	for i := len(stack) - 1; i >= 0; i-- {
		w := stack[i]
		if w == sourceID {
			continue
		}

		for _, v := range pred[w] {
			if sigma[w] > 0 {
				delta[v] += (sigma[v] / sigma[w]) * (1 + delta[w])
			}
		}

		if w != sourceID {
			bc[w] += delta[w]
		}
	}
}

// RecommendSampleSize returns a recommended sample size based on graph characteristics.
// The goal is to balance accuracy vs. speed.
func RecommendSampleSize(nodeCount, edgeCount int) int {
	switch {
	case nodeCount < 100:
		// Small graph: use exact algorithm
		return nodeCount
	case nodeCount < 500:
		// Medium graph: 20% sample for ~22% error
		minSample := 50
		sample := nodeCount / 5
		if sample > minSample {
			return sample
		}
		return minSample
	case nodeCount < 2000:
		// Large graph: fixed sample for ~10% error
		return 100
	default:
		// XL graph: larger fixed sample
		return 200
	}
}
