import { useEffect } from 'preact/hooks';
import { currentPath } from './state';
import { HomePage } from './components/HomePage/HomePage';
import { RoomPage } from './components/RoomPage/RoomPage';
import { Footer } from './components/Footer/Footer';
import { Toast } from './components/Toast/Toast';

export function App() {
  // Listen for browser back/forward navigation
  useEffect(() => {
    function handlePopState() {
      currentPath.value = window.location.pathname;
    }
    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  const path = currentPath.value;

  // Simple router: / or /room/:id
  let page;
  if (path.startsWith('/room/')) {
    page = <RoomPage path={path} />;
  } else {
    page = <HomePage />;
  }

  return (
    <>
      {page}
      <Footer />
      <Toast />
    </>
  );
}
