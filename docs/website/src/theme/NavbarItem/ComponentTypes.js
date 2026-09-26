import ComponentTypes from '@theme-original/NavbarItem/ComponentTypes';
import VersionSwitcher from '@site/src/components/VersionSwitcher';

// Adds `type: 'custom-versionSwitcher'` for docusaurus.config.js's navbar.
export default {
  ...ComponentTypes,
  'custom-versionSwitcher': VersionSwitcher,
};
