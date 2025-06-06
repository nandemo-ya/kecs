import React from 'react';

export const BrowserRouter = ({ children }: { children: React.ReactNode }) => <>{children}</>;
export const Routes = ({ children }: { children: React.ReactNode }) => <>{children}</>;
export const Route = () => null;
export const Link = ({ children, to }: { children: React.ReactNode; to: string }) => <a href={to}>{children}</a>;
export const useNavigate = () => jest.fn();
export const useParams = () => ({});
export const useLocation = () => ({ pathname: '/' });