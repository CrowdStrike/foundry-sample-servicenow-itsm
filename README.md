![CrowdStrike Falcon](/.doc_assets/images/images/cs-logo.png?raw=true)

# ServiceNow ITSM Helper Foundry App

The ServiceNow ITSM Helper is a community-driven, open source project which serves as an example of an app which can be built using CrowdStrike's Foundry ecosystem. `foundry-servicenow-itsm-helper` is an open source project, not a CrowdStrike product. As such, it carries no formal support, expressed or implied.

This app is one of several App Templates included in Foundry that you can use to jumpstart your development. It comes complete with a set of preconfigured capabilities aligned to its business purpose. Deploy this app from the Templates page with a single click in the Foundry UI, or create an app from this template using the CLI.

> [!IMPORTANT]  
> To view documentation and deploy this app, you need access to the Falcon console.

## Description

The ServiceNow ITSM Helper is a Foundry application that enables seamless integration between CrowdStrike Falcon and ServiceNow ITSM systems. This integration allows security teams to efficiently manage incidents by automatically creating and updating ServiceNow tickets based on CrowdStrike detections and alerts.

### Key Capabilities and Features

- **One-Way Alert Synchronization**: Automatically create ServiceNow incidents from CrowdStrike alerts
- **Entity Mapping**: Track relationships between CrowdStrike entities and ServiceNow tickets
- **Support for Multiple Ticket Types**: Create standard incidents or Security Incident Response (SIR) tickets
- **Ticket as a Container**: Associate multiple security objects (hosts, users, alerts) with a single ServiceNow ticket
- **Time-Based Throttling**: Control the flow of updates to prevent duplicate tickets and unnecessary noise
- **Customizable Fields**: Map CrowdStrike data to ServiceNow fields with support for custom fields and hierarchical categories

### Additional Requirements

- For SIR ticket functionality: ServiceNow Security Incident Response module
- API access to your ServiceNow instance
- Appropriate permissions in both CrowdStrike and ServiceNow

## Prerequisites

* The Foundry CLI (instructions below).
* Golang (needed if modifying the app's functions).
* A ServiceNow instance with ITSM and/or SIR module installed.

### Install the Foundry CLI

You can install the Foundry CLI with Scoop on Windows or Homebrew on Linux/macOS.

**Windows**:

Install [Scoop](https://scoop.sh/). Then, add the Foundry CLI bucket and install the Foundry CLI.

```shell
scoop bucket add foundry https://github.com/crowdstrike/scoop-foundry-cli.git
scoop install foundry
```

Or, you can download the [latest Windows zip file](https://assets.foundry.crowdstrike.com/cli/latest/foundry_Windows_x86_64.zip), expand it, and add the install directory to your PATH environment variable.

**Linux and macOS**:

Install [Homebrew](https://docs.brew.sh/Installation). Then, add the Foundry CLI repository to the list of formulae that Homebrew uses and install the CLI:

```shell
brew tap crowdstrike/foundry-cli
brew install crowdstrike/foundry-cli/foundry
```

Run `foundry version` to verify it's installed correctly.

## Getting Started

Clone this sample to your local system, or [download as a zip file](https://github.com/CrowdStrike/foundry-servicenow-itsm-helper/archive/refs/heads/main.zip) and import it into Foundry. 

```shell
git clone https://github.com/CrowdStrike/foundry-servicenow-itsm-helper
cd foundry-servicenow-itsm-helper
```

Log in to Foundry:

```shell
foundry login
```

Select the following permissions:

- [ ] Create and run RTR scripts
- [x] Create, execute and test workflow templates
- [ ] Create, run and view API integrations
- [ ] Create, edit, delete, and list queries

Deploy the app:

```shell
foundry apps deploy
```

> [!TIP]
> If you get an error that the name already exists, change the name to something unique to your CID in `manifest.yml`.

Once the deployment has finished, you can release the app:

```shell
foundry apps release
```

Next, go to **Foundry** > **App catalog**, find your app, and install it.

## About this app

The ServiceNow ITSM Helper Foundry app automates the integration between ServiceNow and CrowdStrike's Fusion SOAR platform, streamlining IT Service Management workflows.

For more information about this app:
- [User Documentation](USERDOCS.md) - Detailed user guide with setup instructions, use cases, and configuration options
- [Components Documentation](COMPONENTS.md) - Technical documentation of the app's components, actions, and integrations

## Foundry resources

- Foundry documentation: [US-1](https://falcon.crowdstrike.com/documentation/category/c3d64B8e/falcon-foundry) | [US-2](https://falcon.us-2.crowdstrike.com/documentation/category/c3d64B8e/falcon-foundry) | [EU](https://falcon.eu-1.crowdstrike.com/documentation/category/c3d64B8e/falcon-foundry)
- Foundry learning resources: [US-1](https://falcon.crowdstrike.com/foundry/learn) | [US-2](https://falcon.us-2.crowdstrike.com/foundry/learn) | [EU](https://falcon.eu-1.crowdstrike.com/foundry/learn)

---

<p align="center"><img src="/.doc_assets/images/project/cs-logo-footer.png"><br/><img width="300px" src="/.doc_assets/images/project/adversary-goblin-panda.png"></p>
<h3><p align="center">WE STOP BREACHES</p></h3>
