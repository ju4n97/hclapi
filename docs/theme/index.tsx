import { Layout as BasicLayout } from '@rspress/core/theme-original';

const Layout = () => (
  <BasicLayout
    afterNavMenu={
      <div className="hclapi-nav-actions">    
        <a
          href="/installation"
          className="hclapi-button hclapi-button--primary"
        >
          Install
        </a>
      </div>
    }
  />
);

export * from '@rspress/core/theme-original';
export { Layout };
