# Namespace Management

This guide explains how the Workload Variant Autoscaler (WVA) controller manages namespaces and how you can control which namespaces are watched and reconciled.

## Overview

The WVA controller supports flexible namespace management through:

1. **Multi-namespace mode** (default): Watch all namespaces with optional exclusions
2. **Single-namespace mode**: Watch only a specific namespace
3. **Namespace exclusion**: Exclude specific namespaces from management
4. **Namespace opt-in**: Explicitly enable namespace-local ConfigMap watching

## Operating Modes

### Multi-Namespace Mode (Default)

By default, the controller watches **all namespaces** in the cluster for:
- VariantAutoscaling resources
- Namespace-local ConfigMaps (in tracked namespaces)

This is the default for the kustomize deployment (`config/default`). No flags are required. To restrict the controller to a single namespace, use `--watch-namespace` (see below).

**Characteristics:**
- Watches all namespaces by default
- Respects namespace exclusion annotations
- Scales VariantAutoscaling resources across the cluster
- Allows namespace-local configuration overrides

**Use cases:**
- Single controller managing the entire cluster
- Centralized autoscaling management
- Multi-tenant clusters with selective exclusions

---

### Single-Namespace Mode

In single-namespace mode, the controller watches **only one specific namespace** specified via the `--watch-namespace` CLI flag.

**Enable single-namespace mode:**

Add the `--watch-namespace` flag to the controller deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: workload-variant-autoscaler-controller-manager
  namespace: wva-system
spec:
  template:
    spec:
      containers:
      - name: manager
        args:
        - --watch-namespace=my-namespace
        # ... other args ...
```

Or set the `WATCH_NAMESPACE` environment variable:

```yaml
env:
- name: WATCH_NAMESPACE
  value: "my-namespace"
```

**Behavior in single-namespace mode:**
- **ONLY** the specified namespace is watched
- Namespace exclusion annotation (`wva.llmd.ai/exclude`) is **ignored** for the watched namespace
- Namespace opt-in label (`wva.llmd.ai/config-enabled`) is **ignored** for the watched namespace
- All VariantAutoscaling resources in the watched namespace are reconciled
- All namespace-local ConfigMaps in the watched namespace are processed

**Use cases:**
- Multi-tenant clusters where each tenant has a dedicated controller instance
- Restricted RBAC scenarios (controller has permissions only for one namespace)
- Development and testing environments
- Namespace-level isolation requirements

**Important:** The explicit CLI flag takes precedence over all annotation/label-based filtering. This ensures that when an operator explicitly specifies a namespace, the controller honors that decision.

---

## Namespace Filtering

### Exclusion Annotation

You can exclude specific namespaces from WVA management by adding the `wva.llmd.ai/exclude: "true"` annotation to the namespace.

**Exclude a namespace:**

```bash
kubectl annotate namespace my-namespace wva.llmd.ai/exclude=true
```

**Example:**

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: my-namespace
  annotations:
    wva.llmd.ai/exclude: "true"
```

**What gets excluded:**
- VariantAutoscaling resources in the namespace will **NOT** be reconciled
- Namespace-local ConfigMaps will **NOT** be watched

**Use cases:**
- System namespaces that should not be managed (e.g., `kube-system`, `kube-public`)
- Namespaces with custom scaling logic
- Temporary exclusion during maintenance

**Remove exclusion:**

```bash
kubectl annotate namespace my-namespace wva.llmd.ai/exclude-
```

**Important:** The exclusion annotation is **ignored** in single-namespace mode. If you run the controller with `--watch-namespace=my-namespace`, it will manage that namespace even if it has the exclusion annotation.

---

### Opt-In Label for ConfigMaps

By default, namespace-local ConfigMaps are only watched in namespaces that have at least one VariantAutoscaling resource. This prevents unnecessary cluster-wide watching.

However, you can explicitly enable ConfigMap watching in a namespace using the opt-in label, even if there are no VariantAutoscaling resources yet.

**Enable ConfigMap watching:**

```bash
kubectl label namespace my-namespace wva.llmd.ai/config-enabled=true
```

**Example:**

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: my-namespace
  labels:
    wva.llmd.ai/config-enabled: "true"
```

**When to use opt-in:**
- You want to pre-configure saturation settings before creating VAs
- You're using namespace-local scale-to-zero configuration without VAs
- You want explicit control over which namespaces support local configuration

**Automatic tracking:** When you create a VariantAutoscaling resource in a namespace, that namespace is automatically tracked for ConfigMap watching. The opt-in label is optional in this case.

---

## Namespace-Local Configuration

The controller supports namespace-local configuration overrides through ConfigMaps. This allows different namespaces to have different saturation thresholds or scale-to-zero settings.

### Supported ConfigMaps

| ConfigMap Name | Purpose |
|----------------|---------|
| `wva-saturation-config` | Namespace-local saturation configuration |
| `wva-scale-to-zero-config` | Namespace-local scale-to-zero configuration |

### Configuration Hierarchy

Configuration is resolved in the following order (highest to lowest priority):

1. **Namespace-local ConfigMap** (if exists and namespace is tracked)
2. **Global ConfigMap** (in controller namespace)
3. **Controller defaults**

### Example: Namespace-Local Saturation Config

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: wva-saturation-config
  namespace: my-namespace
data:
  default: |
    kvCacheThreshold: 0.85
    queueLengthThreshold: 15
    kvSpareTrigger: 0.10
    queueSpareTrigger: 5
```

This ConfigMap will **only** be used if:
- The namespace is **not excluded** (`wva.llmd.ai/exclude != "true"`), OR
- The controller is running in single-namespace mode watching this namespace

AND one of:
- The namespace has at least one VariantAutoscaling resource (automatic tracking), OR
- The namespace has the opt-in label (`wva.llmd.ai/config-enabled: "true"`), OR
- The controller is running in single-namespace mode watching this namespace

For more information on configuration, see the [Configuration Guide](configuration.md).

---

## Multi-Controller Isolation

If you're running multiple WVA controller instances (for high availability or multi-tenancy), you can use the `CONTROLLER_INSTANCE` environment variable and corresponding label to isolate controllers.

See the [Multi-Controller Isolation Guide](multi-controller-isolation.md) for details.

---

## Common Scenarios

### Scenario 1: Exclude System Namespaces

Prevent the controller from managing system namespaces:

```bash
kubectl annotate namespace kube-system wva.llmd.ai/exclude=true
kubectl annotate namespace kube-public wva.llmd.ai/exclude=true
kubectl annotate namespace kube-node-lease wva.llmd.ai/exclude=true
```

---

### Scenario 2: Multi-Tenant Cluster with Namespace Isolation

Run one controller instance per tenant namespace:

**Tenant A controller:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: wva-controller-tenant-a
  namespace: wva-system
spec:
  template:
    spec:
      containers:
      - name: manager
        args:
        - --watch-namespace=tenant-a
        env:
        - name: CONTROLLER_INSTANCE
          value: "tenant-a"
```

**Tenant B controller:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: wva-controller-tenant-b
  namespace: wva-system
spec:
  template:
    spec:
      containers:
      - name: manager
        args:
        - --watch-namespace=tenant-b
        env:
        - name: CONTROLLER_INSTANCE
          value: "tenant-b"
```

Each controller only manages its assigned namespace.

---

### Scenario 3: Pre-Configure Namespace Before Creating VAs

Enable ConfigMap watching and create namespace-local configuration before VAs exist:

```bash
# 1. Create namespace
kubectl create namespace my-app

# 2. Enable ConfigMap watching
kubectl label namespace my-app wva.llmd.ai/config-enabled=true

# 3. Create namespace-local saturation config
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: wva-saturation-config
  namespace: my-app
data:
  default: |
    kvCacheThreshold: 0.80
    queueLengthThreshold: 10
EOF

# 4. Later, create VAs in the namespace
# They will automatically use the namespace-local configuration
```

---

### Scenario 4: Temporary Exclusion During Maintenance

Exclude a namespace temporarily to prevent autoscaling during maintenance:

```bash
# Before maintenance
kubectl annotate namespace my-app wva.llmd.ai/exclude=true

# Perform maintenance...

# After maintenance
kubectl annotate namespace my-app wva.llmd.ai/exclude-
```

---

## Behavioral Summary

### Multi-Namespace Mode

| Namespace Condition | VAs Reconciled? | ConfigMaps Watched? |
|---------------------|-----------------|---------------------|
| No annotation/label | ✅ Yes | ✅ Yes (if tracked or opt-in) |
| `exclude: "true"` | ❌ No | ❌ No |
| `config-enabled: "true"` | ✅ Yes | ✅ Yes |
| Has VA resources | ✅ Yes | ✅ Yes (auto-tracked) |

### Single-Namespace Mode

| Namespace | VAs Reconciled? | ConfigMaps Watched? |
|-----------|-----------------|---------------------|
| Watched namespace | ✅ Yes (ignores exclusion) | ✅ Yes (ignores exclusion and opt-in) |
| Other namespaces | ❌ No | ❌ No |

---

## Troubleshooting

If your namespaces aren't behaving as expected, see the [Troubleshooting Guide](troubleshooting.md) for help diagnosing namespace-related issues.

Common issues:
- [Why is my VA not being reconciled?](troubleshooting.md#why-is-my-variantautoscaling-not-being-reconciled)
- [Why is my namespace-local ConfigMap not being used?](troubleshooting.md#why-is-my-namespace-local-configmap-not-being-used)

---

## Best Practices

1. **Exclude system namespaces:** Always exclude `kube-system`, `kube-public`, and other system namespaces to prevent accidental management.

2. **Use single-namespace mode for multi-tenancy:** In multi-tenant environments, run dedicated controller instances per tenant with `--watch-namespace` to ensure complete isolation.

3. **Prefer automatic tracking:** Let the controller automatically track namespaces when VAs are created, rather than using opt-in labels everywhere.

4. **Document exclusions:** Maintain documentation of which namespaces are excluded and why, to avoid confusion during troubleshooting.

5. **Test namespace configuration:** Before rolling out namespace-local configuration, test in a development namespace first.

---

## Related Documentation

- [Configuration Guide](configuration.md)
- [Multi-Controller Isolation](multi-controller-isolation.md)
- [Troubleshooting Guide](troubleshooting.md)
- [Installation Guide](installation.md)
