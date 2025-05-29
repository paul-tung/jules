import { Routes } from '@angular/router';
import { LoginComponent } from './components/login/login';
import { RegisterComponent } from './components/register/register';
import { DashboardComponent } from './components/dashboard/dashboard';
import { VideoUploadComponent } from './components/video-upload/video-upload';
import { VideoListComponent } from './components/video-list/video-list'; // Added
import { VideoEditComponent } from './components/video-edit/video-edit'; // Added
import { AuthGuard } from './guards/auth-guard'; 
import { inject } from '@angular/core'; 

export const routes: Routes = [
  { path: 'login', component: LoginComponent },
  { path: 'register', component: RegisterComponent },
  {
    path: 'dashboard',
    component: DashboardComponent,
    canActivate: [(route, state) => inject(AuthGuard).canActivate(route, state)]
  },
  {
    path: 'upload-video', 
    component: VideoUploadComponent,
    canActivate: [(route, state) => inject(AuthGuard).canActivate(route, state)]
  },
  {
    path: 'videos', // Route for listing videos
    component: VideoListComponent,
    canActivate: [(route, state) => inject(AuthGuard).canActivate(route, state)]
  },
  {
    path: 'videos/:videoId/edit', // Route for editing a specific video
    component: VideoEditComponent,
    canActivate: [(route, state) => inject(AuthGuard).canActivate(route, state)]
  },
  { path: '', redirectTo: '/login', pathMatch: 'full' }, 
  { path: '**', redirectTo: '/login' } 
];
