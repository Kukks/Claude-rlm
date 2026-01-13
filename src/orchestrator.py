#!/usr/bin/env python3
"""
RLM Orchestrator - Trampoline-based recursive document analysis

This module implements the core orchestration logic for the RLM plugin,
enabling recursive document analysis through a trampoline pattern that
respects Claude Code's "subagents can't spawn subagents" constraint.
"""
import json
import os
from typing import Dict, Any, Optional, List
from dataclasses import dataclass, asdict
from datetime import datetime
import hashlib

@dataclass
class ContinuationRequest:
    """Request from subagent to recurse deeper"""
    agent_type: str  # "Explorer", "Worker", "Synthesizer"
    task: str
    context: Dict[str, Any]
    return_to: str  # Identifier for result storage
    metadata: Optional[Dict[str, Any]] = None

    def to_dict(self):
        return asdict(self)

    @classmethod
    def from_dict(cls, data: dict):
        return cls(**data)

@dataclass
class AnalysisResult:
    """Completed analysis from subagent"""
    content: str
    metadata: Dict[str, Any]
    token_count: int
    cost_usd: float

    def to_dict(self):
        return asdict(self)

@dataclass
class Task:
    """Represents a task for a subagent"""
    agent_type: str
    task_description: str
    context: Dict[str, Any]
    depth: int
    return_to: Optional[str] = None
    child_results: Optional[Dict[str, Any]] = None

    def to_dict(self):
        return asdict(self)

    @classmethod
    def from_dict(cls, data: dict):
        return cls(**data)

class RLMOrchestrator:
    """Manages trampoline-based recursive document analysis"""

    def __init__(self, config: Optional[Dict] = None):
        self.config = config or self.load_config()
        self.state_file = ".rlm_state.json"
        self.results_dir = "analysis"
        self.cache_dir = ".rlm_cache"

        # Create directories
        os.makedirs(self.results_dir, exist_ok=True)
        os.makedirs(self.cache_dir, exist_ok=True)

        # Statistics
        self.stats = {
            "total_subagent_calls": 0,
            "total_tokens": 0,
            "total_cost_usd": 0.0,
            "max_depth_reached": 0,
            "cache_hits": 0,
            "start_time": datetime.now().isoformat()
        }

    def load_config(self) -> Dict:
        """Load plugin configuration"""
        config_path = os.path.expanduser("~/.claude-rlm/config.json")
        if os.path.exists(config_path):
            with open(config_path) as f:
                return json.load(f)

        # Default config
        return {
            "max_recursion_depth": 10,
            "cache_enabled": True,
            "cache_ttl_hours": 24,
            "parallel_branches": 3,
            "cost_tracking": True
        }

    def load_state(self) -> Optional[Dict]:
        """Load orchestrator state"""
        if os.path.exists(self.state_file):
            with open(self.state_file) as f:
                data = json.load(f)
                # Convert task dicts back to Task objects
                data['stack'] = [Task.from_dict(t) for t in data['stack']]
                data['current_task'] = Task.from_dict(data['current_task'])
                return data

        return None

    def save_state(self, stack: List[Task], current_task: Task, results: Dict):
        """Persist orchestrator state"""
        state = {
            "stack": [t.to_dict() for t in stack],
            "current_task": current_task.to_dict(),
            "results": results,
            "stats": self.stats,
            "timestamp": datetime.now().isoformat()
        }

        with open(self.state_file, 'w') as f:
            json.dump(state, f, indent=2)

    def clear_state(self):
        """Clear state after completion"""
        if os.path.exists(self.state_file):
            os.remove(self.state_file)

    def get_cache_key(self, task: Task) -> str:
        """Generate cache key for task"""
        key_data = f"{task.agent_type}:{task.task_description}:{json.dumps(task.context, sort_keys=True)}"
        return hashlib.sha256(key_data.encode()).hexdigest()

    def check_cache(self, task: Task) -> Optional[AnalysisResult]:
        """Check if result is cached"""
        if not self.config["cache_enabled"]:
            return None

        cache_key = self.get_cache_key(task)
        cache_path = os.path.join(self.cache_dir, f"{cache_key}.json")

        if os.path.exists(cache_path):
            # Check TTL
            mtime = os.path.getmtime(cache_path)
            age_hours = (datetime.now().timestamp() - mtime) / 3600

            if age_hours < self.config["cache_ttl_hours"]:
                with open(cache_path) as f:
                    data = json.load(f)
                    self.stats["cache_hits"] += 1
                    return AnalysisResult(**data)

        return None

    def save_cache(self, task: Task, result: AnalysisResult):
        """Save result to cache"""
        if not self.config["cache_enabled"]:
            return

        cache_key = self.get_cache_key(task)
        cache_path = os.path.join(self.cache_dir, f"{cache_key}.json")

        with open(cache_path, 'w') as f:
            json.dump(result.to_dict(), f, indent=2)

    def dispatch_subagent(self, task: Task) -> Any:
        """
        Dispatch task to appropriate subagent

        NOTE: This is where Claude Code's native subagent spawning happens.
        The actual implementation will use Claude's Task tool or Agent SDK.
        This is a placeholder showing the interface.
        """
        # Check cache first
        cached = self.check_cache(task)
        if cached:
            print(f"✅ Cache hit for {task.agent_type} task")
            return cached

        # Update stats
        self.stats["total_subagent_calls"] += 1
        self.stats["max_depth_reached"] = max(self.stats["max_depth_reached"], task.depth)

        print(f"\n🎯 Spawning {task.agent_type} subagent (depth {task.depth})")
        print(f"   Task: {task.task_description[:80]}...")

        # Format task for subagent
        subagent_prompt = self.format_task_for_subagent(task)

        # This would be replaced with actual Claude Code subagent spawning:
        # result = spawn_subagent(task.agent_type, subagent_prompt)

        # For now, return placeholder that shows expected structure
        print(f"   [Subagent would execute here]")
        return {
            "type": "RESULT",
            "content": f"Analysis of {task.task_description}",
            "metadata": {"task": task.to_dict()},
            "token_count": 1000,
            "cost_usd": 0.003
        }

    def format_task_for_subagent(self, task: Task) -> str:
        """Format task into subagent prompt"""
        prompt_parts = [
            f"You are a {task.agent_type} subagent in a recursive document analysis system.",
            f"\nTask: {task.task_description}",
            f"\nDepth: {task.depth}/{self.config['max_recursion_depth']}",
        ]

        if task.context:
            prompt_parts.append(f"\nContext:\n{json.dumps(task.context, indent=2)}")

        if task.child_results:
            prompt_parts.append(f"\nChild Results:\n{json.dumps(task.child_results, indent=2)}")

        prompt_parts.append("\n---")
        prompt_parts.append("\nIf you need deeper analysis, return a ContinuationRequest:")
        prompt_parts.append("""
```json
{
  "type": "CONTINUATION",
  "agent_type": "Worker",
  "task": "Detailed description",
  "context": {"key": "value"},
  "return_to": "unique_identifier"
}
```
""")
        prompt_parts.append("\nOtherwise, return your analysis results directly.")

        return "\n".join(prompt_parts)

    def is_continuation(self, result: Any) -> bool:
        """Check if result is a continuation request"""
        if isinstance(result, dict):
            return result.get("type") == "CONTINUATION"
        return False

    def create_task_from_continuation(self, parent_task: Task, cont: Dict) -> Task:
        """Create new task from continuation request"""
        return Task(
            agent_type=cont["agent_type"],
            task_description=cont["task"],
            context=cont["context"],
            depth=parent_task.depth + 1,
            return_to=cont["return_to"]
        )

    def analyze_document(self, document_path: str, query: str) -> AnalysisResult:
        """
        Main entry point for document analysis

        Args:
            document_path: Path to document or directory to analyze
            query: Analysis objective

        Returns:
            Final analysis result
        """
        # Check for existing state (resume capability)
        existing_state = self.load_state()

        if existing_state:
            print("🔄 Resuming from saved state...")
            stack = existing_state["stack"]
            current_task = existing_state["current_task"]
            results = existing_state["results"]
            self.stats = existing_state.get("stats", self.stats)
        else:
            # Initialize new analysis
            print("🚀 Starting new RLM analysis...")
            stack = []
            current_task = Task(
                agent_type="Explorer",
                task_description=f"Analyze document at {document_path} for: {query}",
                context={
                    "document_path": document_path,
                    "query": query,
                    "analysis_start": datetime.now().isoformat()
                },
                depth=0
            )
            results = {}

        # Trampoline loop
        iteration = 0
        while True:
            iteration += 1

            # Safety check
            if iteration > 1000:
                raise RuntimeError("Exceeded maximum iterations (possible infinite loop)")

            if current_task.depth > self.config["max_recursion_depth"]:
                raise RuntimeError(f"Exceeded max recursion depth: {self.config['max_recursion_depth']}")

            # Execute current task
            result = self.dispatch_subagent(current_task)

            # Update cost tracking
            if isinstance(result, dict) and "cost_usd" in result:
                self.stats["total_cost_usd"] += result["cost_usd"]
                self.stats["total_tokens"] += result.get("token_count", 0)

            # Check if subagent wants to recurse
            if self.is_continuation(result):
                print(f"🔄 Subagent requests continuation: {result['agent_type']}")

                # Push current task to stack
                stack.append(current_task)

                # Create new task from continuation
                current_task = self.create_task_from_continuation(current_task, result)

                # Save state
                self.save_state(stack, current_task, results)

                print(f"📊 Stack depth: {len(stack)}, Current depth: {current_task.depth}")
                continue  # Trampoline!

            # Result completed - do we have a parent waiting?
            if stack:
                print(f"✅ Subagent completed, returning to parent")

                # Store result
                if current_task.return_to:
                    results[current_task.return_to] = result

                # Cache the result
                if isinstance(result, AnalysisResult):
                    self.save_cache(current_task, result)

                # Pop parent task
                parent_task = stack.pop()

                # Inject child results
                parent_task.child_results = results.copy()

                current_task = parent_task

                # Save state
                self.save_state(stack, current_task, results)

                print(f"🔙 Resumed parent at depth {current_task.depth}")
                continue

            # Stack empty - we're done!
            print(f"\n🎉 Analysis complete!")
            print(f"📊 Stats:")
            print(f"   Total subagent calls: {self.stats['total_subagent_calls']}")
            print(f"   Max depth reached: {self.stats['max_depth_reached']}")
            print(f"   Cache hits: {self.stats['cache_hits']}")
            print(f"   Total tokens: {self.stats['total_tokens']:,}")
            print(f"   Total cost: ${self.stats['total_cost_usd']:.4f}")

            # Clear state
            self.clear_state()

            return result

    def get_status(self) -> Dict:
        """Get current analysis status"""
        state = self.load_state()

        if not state:
            return {"status": "idle", "message": "No active analysis"}

        return {
            "status": "active",
            "current_depth": state["current_task"].depth,
            "stack_size": len(state["stack"]),
            "results_count": len(state["results"]),
            "stats": state["stats"]
        }

# CLI interface for testing
if __name__ == "__main__":
    import sys

    if len(sys.argv) < 2:
        print("Usage: python orchestrator.py <document_path> <query>")
        print("   or: python orchestrator.py status")
        sys.exit(1)

    orchestrator = RLMOrchestrator()

    if sys.argv[1] == "status":
        status = orchestrator.get_status()
        print(json.dumps(status, indent=2))
    else:
        document_path = sys.argv[1]
        query = " ".join(sys.argv[2:]) if len(sys.argv) > 2 else "Analyze this document"

        result = orchestrator.analyze_document(document_path, query)
        print("\n" + "="*80)
        print("FINAL RESULT")
        print("="*80)
        print(json.dumps(result, indent=2))
