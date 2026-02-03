---
sidebar_position: 8
---

import CodeBlock from '@theme/CodeBlock';
import StanzaS3 from '!!raw-loader!../../examples/stanza.yaml';
import StanzaAzure from '!!raw-loader!../../examples/stanza_azure.yaml';

# Supported repositories type

The pgBackRest plugin enables PostgreSQL backup files to be stored in:

- Amazon s3
- Microsoft Azure Blob Storage

The plugin relies on the protocols that pgBackRest supports natively. To
configure the repositories for pgBackRest, you must define a `Stanza`
object, which establishes the link between your PostgreSQL `Cluster` and
the repository or repositories.

Below are a few examples of how to use the supported backup storage
backend.

## Amazon S3, or S3 compatible solutions

<CodeBlock language="yaml">{StanzaS3}</CodeBlock>

## Azure Blob Storage

<CodeBlock language="yaml">{StanzaAzure}</CodeBlock>

<!--
    vim: spelllang=en spell
  -->
