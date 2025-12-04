# Main Terraform configuration for GKE cluster with NGINX Ingress

terraform {
  required_version = ">= 1.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23"
    }
  }
}

# Configure the Google Cloud Provider
provider "google" {
  project = var.project_id
  region  = var.region
}

# Get GKE cluster credentials for Kubernetes/Helm providers
data "google_client_config" "default" {}

# Configure Kubernetes Provider
provider "kubernetes" {
  host                   = "https://${google_container_cluster.primary.endpoint}"
  token                  = data.google_client_config.default.access_token
  cluster_ca_certificate = base64decode(google_container_cluster.primary.master_auth[0].cluster_ca_certificate)
}

# Create GKE cluster
resource "google_container_cluster" "primary" {
  name                     = var.cluster_name
  location                 = var.zone != "" ? var.zone : var.region
  remove_default_node_pool = false
  deletion_protection      = false

  network           = google_compute_network.vpc.self_link
  datapath_provider = "ADVANCED_DATAPATH"

  min_master_version = var.kubernetes_version

  cluster_autoscaling {
    enabled             = var.enable_autoscaling
    autoscaling_profile = var.autoscaling_profile
    resource_limits {
      resource_type = "cpu"
      minimum       = (var.min_node_count * 2) + (var.vm_min_node_count * 2)
      maximum       = (var.max_node_count * 4) + (var.vm_max_node_count * 4)
    }
    resource_limits {
      resource_type = "memory"
      minimum       = (var.min_node_count * 8) + (var.vm_min_node_count * 16)
      maximum       = (var.max_node_count * 8) + (var.vm_max_node_count * 16)
    }
    auto_provisioning_defaults {
      service_account = google_service_account.kubernetes.email
      management {
        auto_repair  = true
        auto_upgrade = true
      }
      disk_size        = 50
      disk_type        = "pd-standard"
      min_cpu_platform = "Intel Broadwell"
    }
  }

  node_pool {
    name       = "default-nodes"
    node_count = var.min_node_count

    autoscaling {
      min_node_count  = var.min_node_count
      max_node_count  = var.max_node_count
      location_policy = "ANY"
    }

    management {
      auto_repair  = true
      auto_upgrade = true
    }

    node_config {
      preemptible  = true
      machine_type = var.machine_type
      disk_size_gb = var.disk_size_gb
      disk_type    = "pd-standard"
      image_type   = "UBUNTU_CONTAINERD"

      service_account = google_service_account.kubernetes.email

      labels = {
        node_pool = "default-pool"
      }
    }
  }

  node_pool {
    name = "vm-node-pool"
    # Note: inline pools use 'node_count', not 'initial_node_count'
    # You might need to adjust this based on your variable
    node_count = var.vm_min_node_count

    autoscaling {
      min_node_count  = var.vm_min_node_count
      max_node_count  = var.vm_max_node_count
      location_policy = "ANY"
    }

    management {
      auto_repair  = true
      auto_upgrade = true
    }

    node_config {
      preemptible     = var.vm_preemptible_nodes
      machine_type    = var.vm_machine_type
      disk_size_gb    = var.vm_disk_size_gb
      disk_type       = "pd-standard"
      image_type      = "UBUNTU_CONTAINERD"
      service_account = google_service_account.kubernetes.email
      oauth_scopes = [
        "https://www.googleapis.com/auth/cloud-platform"
      ]
      labels = {
        app       = "vm"
        node_pool = "vm-pool"
      }
      taint {
        key    = "vm"
        value  = "true"
        effect = "NO_SCHEDULE"
      }
    }

    upgrade_settings {
      max_surge       = 1
      max_unavailable = 0
      strategy        = "SURGE"
    }
  }
}

# Ingress firewall rule
resource "google_compute_firewall" "nginx_ingress" {
  name    = "${var.cluster_name}-nginx-ingress"
  network = google_compute_network.vpc.name
  project = var.project_id

  allow {
    protocol = "tcp"
    ports    = ["80", "443"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["gke-node", var.cluster_name]
}

# Create a service account for the cluster
resource "google_service_account" "kubernetes" {
  account_id   = "${var.cluster_name}-sa"
  display_name = "Service Account for ${var.cluster_name} GKE cluster"
  project      = var.project_id
}

# Create VPC network
resource "google_compute_network" "vpc" {
  name                    = "${var.cluster_name}-vpc"
  auto_create_subnetworks = true
  project                 = var.project_id
}
