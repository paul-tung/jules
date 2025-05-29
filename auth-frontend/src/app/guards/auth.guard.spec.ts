import { TestBed } from '@angular/core/testing';
import { Router, UrlTree } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { Observable, of } from 'rxjs';

import { AuthGuard } from './auth-guard'; // Use correct path
import { AuthService } from '../services/auth'; // Use correct path

describe('AuthGuard', () => {
  let guard: AuthGuard;
  let mockAuthService: jasmine.SpyObj<AuthService>;
  let router: Router;
  let mockRouter: jasmine.SpyObj<Router>;


  // Mock ActivatedRouteSnapshot and RouterStateSnapshot as they are simple objects
  const dummyRoute = {} as import('@angular/router').ActivatedRouteSnapshot;
  const dummyState = {} as import('@angular/router').RouterStateSnapshot;


  beforeEach(() => {
    mockAuthService = jasmine.createSpyObj('AuthService', ['isAuthenticatedUser']);
    mockRouter = jasmine.createSpyObj('Router', ['createUrlTree']);

    TestBed.configureTestingModule({
      imports: [
        RouterTestingModule // Provides a real Router instance if needed, or we can spy
      ],
      providers: [
        AuthGuard,
        { provide: AuthService, useValue: mockAuthService },
        { provide: Router, useValue: mockRouter } // Use the spy for Router
      ]
    });
    guard = TestBed.inject(AuthGuard);
    router = TestBed.inject(Router); // This will be the mockRouter
  });

  it('should be created', () => {
    expect(guard).toBeTruthy();
  });

  it('should allow activation if user is authenticated', () => {
    mockAuthService.isAuthenticatedUser.and.returnValue(true);

    const canActivateResult = guard.canActivate(dummyRoute, dummyState);
    
    // Check the direct boolean return if not Observable/Promise
    if (typeof canActivateResult === 'boolean') {
        expect(canActivateResult).toBeTrue();
    } else if (canActivateResult instanceof Observable) {
        (canActivateResult as Observable<boolean | UrlTree>).subscribe(result => {
            expect(result).toBeTrue();
        });
    } else if (canActivateResult instanceof Promise) {
        Promise.resolve(canActivateResult).then(result => {
            expect(result).toBeTrue();
        });
    }
    expect(mockAuthService.isAuthenticatedUser).toHaveBeenCalled();
  });

  it('should prevent activation and redirect to /login if user is not authenticated', ()_ => {
    mockAuthService.isAuthenticatedUser.and.returnValue(false);
    const expectedUrlTree = new UrlTree(); // Dummy UrlTree, actual one is created by router.createUrlTree
    mockRouter.createUrlTree.and.returnValue(expectedUrlTree);


    const canActivateResult = guard.canActivate(dummyRoute, dummyState);

    if (typeof canActivateResult === 'boolean') {
        expect(canActivateResult).toBeFalse(); // Should not happen based on guard logic
    } else if (canActivateResult instanceof Observable) {
        (canActivateResult as Observable<boolean | UrlTree>).subscribe(result => {
            expect(result).toEqual(expectedUrlTree);
        });
    } else if (canActivateResult instanceof Promise) {
         Promise.resolve(canActivateResult).then(result => {
            expect(result).toEqual(expectedUrlTree);
        });
    } else { // Direct UrlTree
        expect(canActivateResult).toEqual(expectedUrlTree);
    }
    
    expect(mockAuthService.isAuthenticatedUser).toHaveBeenCalled();
    expect(mockRouter.createUrlTree).toHaveBeenCalledWith(['/login']);
  });

});
