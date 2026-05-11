import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { Amplify } from 'aws-amplify'
import amplifyConfig from './auth/amplifyConfig'
import App from './App.tsx'

Amplify.configure(amplifyConfig)  

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
