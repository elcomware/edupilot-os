# Core UI resources

Core owns the EduPilot OS identity, shared layouts, controls, and client-side session utilities. Each feature owns its pages. Each client composes the installed features.

| Location in Core | Responsibility |
|---|---|
| `brand/` | Approved EP marks, wordmark, platform icons and light/dark colour roles |
| `js/layouts/` | Welcome, tenant administrator and platform administrator frames |
| `js/components/`, `css/` | Buttons, cards, tables, dialogs, skeletons and interface styles |
| `native/` | Expo controls, account session lifecycle, feedback and catalog rendering |

Import shared UI through the `@edupilot/core` workspace package, for example `@edupilot/core/web/layouts/WelcomeLayout` or `@edupilot/core/native/ui`. Sibling files can use a local import. The package exports resolve in TypeScript, Vite, Vitest, Node and Metro.

The web host assembles `@edupilot/auth/web` and `@edupilot/tenancy/web` in `clients/web/src/app/pages.ts`. Wails embeds that web build. Expo assembles the native entry points in `clients/mobile/src/pages.ts`. Core receives page data and directory artwork from the host. The server catalog and current grants determine which pages a member can open.

## Brand assets

The vectors, platform exports and the two directory symbols come from the [approved EduPilot brand kit](https://github.com/elcomware/edupilot/tree/0fa69014b92547fcf6efb803eff30ef174531dbd/brand). The outlined wordmark includes the upstream [Inter license](brand/licenses/Inter-OFL.txt). The adjacent OS suffix identifies the application. Use the gold mark and ivory wordmark on the garnet navigation surfaces.

Edit `brand/palette.json` for shared colour roles, then run `node modules/core/resources/brand/build-palette.mjs` from the repository root. Web and Wails load its generated stylesheet. Expo reads the same JSON. The web accent preference applies to interactive controls independently of the product mark.
