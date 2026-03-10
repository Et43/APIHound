<p align="center">
    <picture>
        <img src="cmd/ui/public/img/APIHound_Vertical_RedField.svg" alt="APIHound Community Edition" width='400' />
    </picture>
</p>

<hr />

APIHound is a monolithic web application composed of an embedded React frontend with [Sigma.js](https://www.sigmajs.org/) and a [Go](https://go.dev/) based REST API backend. It is deployed with a [Postgresql](https://www.postgresql.org/) application database and a [Neo4j](https://neo4j.com/) graph database, and is fed by the [SharpHound](https://github.com/SpecterOps/SharpHound) and [AzureHound](https://github.com/SpecterOps/AzureHound) data collectors.

APIHound leverages graph theory to reveal hidden and often unintended relationships across identity and access management systems. Powered by [OpenGraph](https://specterops.io/opengraph/?utm_campaign=Direct_DemoRequest_2025_09_01_GitHub&utm_medium=DemoRequest&utm_source=Direct&Latest_Campaign=701Uw00000X36PF), APIHound now supports comprehensive analysis beyond Active Directory and Azure environments, enabling users to map complex privilege relationships across [diverse identity platforms](https://apihound.specterops.io/opengraph/library). Attackers can utilize APIHound to rapidly discover sophisticated attack paths otherwise impossible to identify manually, while defenders can proactively identify and mitigate these risks. Both red and blue teams benefit from APIHound's expanded capabilities, gaining deeper insights into identity and privilege structures across their entire security landscape.

APIHound CE is created and maintained by the [SpecterOps](https://specterops.io/?utm_campaign=Direct_DemoRequest_2025_09_01_GitHub&utm_medium=DemoRequest&utm_source=Direct&Latest_Campaign=701Uw00000X36PF) team who also brought you [APIHound Enterprise](https://specterops.io/apihound-overview/?utm_campaign=Direct_DemoRequest_2025_09_01_GitHub&utm_medium=DemoRequest&utm_source=Direct&Latest_Campaign=701Uw00000X36PF). The original APIHound was created by [@\_wald0](https://www.twitter.com/_wald0), [@CptJesus](https://twitter.com/CptJesus), and [@harmj0y](https://twitter.com/harmj0y).

## Running APIHound Community Edition
Please refer to the [Quickstart Guide for APIHound Community Edition](https://apihound.specterops.io/get-started/quickstart/community-edition-quickstart), which is part of the [APIHound documentation](https://apihound.specterops.io).

## Useful Links

- [APIHound Documentation](https://apihound.specterops.io/)
- [APIHound Community Edition Quickstart Guide](https://apihound.specterops.io/get-started/quickstart/community-edition-quickstart)
- [APIHound Slack](https://slack.specterops.io)
- [OpenGraph Documentation](https://apihound.specterops.io/opengraph/overview)
- [Wiki](https://github.com/SpecterOps/APIHound/wiki)
- [Docker Compose Example](./examples/docker-compose/README.md)
- [Developer Quick Start Guide](https://github.com/SpecterOps/APIHound/wiki/Development)
- [Contributing Guide](https://github.com/SpecterOps/APIHound/wiki/Contributing)
- [Contributors](./CONTRIBUTORS.md)

## Contact

Please check out the [Contact page](https://github.com/SpecterOps/APIHound/wiki/Contact) in our wiki for details on how to reach out with questions and suggestions.

## Licensing

```
Copyright 2025 Specter Ops, Inc.

Licensed under the Apache License, Version 2.0
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

Unless otherwise annotated by a lower-level LICENSE file or license header, all files in this repository are released
under the `Apache-2.0` license. A full copy of the license may be found in the top-level [LICENSE](LICENSE) file.
