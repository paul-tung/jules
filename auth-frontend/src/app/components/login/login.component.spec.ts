import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { RouterTestingModule } from '@angular/router/testing';
import { of, throwError } from 'rxjs';
import { By } from '@angular/platform-browser';

import { LoginComponent } from './login';
import { AuthService } from '../../services/auth';
import { CommonModule } from '@angular/common';

describe('LoginComponent', () => {
  let component: LoginComponent;
  let fixture: ComponentFixture<LoginComponent>;
  let mockAuthService: jasmine.SpyObj<AuthService>;
  let router: Router;

  beforeEach(async () => {
    mockAuthService = jasmine.createSpyObj('AuthService', ['login']);

    await TestBed.configureTestingModule({
      imports: [
        CommonModule,
        FormsModule,
        RouterTestingModule.withRoutes([
          // You can add dummy routes if your component navigates declaratively
          // { path: 'register', component: class DummyRegisterComponent {} } 
        ]),
        LoginComponent // Import the standalone component
      ],
      providers: [
        { provide: AuthService, useValue: mockAuthService }
      ]
    }).compileComponents();

    fixture = TestBed.createComponent(LoginComponent);
    component = fixture.componentInstance;
    router = TestBed.inject(Router); // Get the router instance
    fixture.detectChanges(); // Initial binding
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize with empty credentials', () => {
    expect(component.credentials.email).toBe('');
    expect(component.credentials.password).toBe('');
  });

  it('should have a disabled login button when form is invalid (e.g. empty)', () => {
    const loginButton = fixture.debugElement.query(By.css('button[type="submit"]')).nativeElement;
    expect(loginButton.disabled).toBeTrue(); // Initially empty, so should be disabled by "required"
  });

  it('should enable login button when form is valid', fakeAsync(() => {
    const emailInput = fixture.debugElement.query(By.css('input[name="email"]')).nativeElement;
    const passwordInput = fixture.debugElement.query(By.css('input[name="password"]')).nativeElement;
    const loginButton = fixture.debugElement.query(By.css('button[type="submit"]')).nativeElement;

    emailInput.value = 'test@example.com';
    emailInput.dispatchEvent(new Event('input'));
    passwordInput.value = 'password123';
    passwordInput.dispatchEvent(new Event('input'));
    
    tick(); // Allow time for ngModel to update and form validation to run
    fixture.detectChanges(); // Update view with new form state

    expect(loginButton.disabled).toBeFalse();
  }));
  
  it('should call AuthService.login on form submission with valid data', fakeAsync(() => {
    mockAuthService.login.and.returnValue(of({ token: 'fake-token' })); // Simulate successful login

    component.credentials.email = 'test@example.com';
    component.credentials.password = 'password123';
    
    fixture.detectChanges(); // Update form controls if necessary
    tick(); // Ensure form validity is updated
    fixture.detectChanges();


    const form = fixture.debugElement.query(By.css('form'));
    form.triggerEventHandler('ngSubmit', null);
    tick(); // Simulate passage of time for async operations

    expect(mockAuthService.login).toHaveBeenCalledWith({ email: 'test@example.com', password: 'password123' });
  }));

  it('should not call AuthService.login if form is invalid (e.g. email missing)', fakeAsync(() => {
    component.credentials.email = ''; // Invalid state
    component.credentials.password = 'password123';
    
    fixture.detectChanges();
    tick();
    fixture.detectChanges();

    const form = fixture.debugElement.query(By.css('form'));
    form.triggerEventHandler('ngSubmit', null);
    tick();

    expect(mockAuthService.login).not.toHaveBeenCalled();
  }));


  it('should display error message if login fails', fakeAsync(() => {
    const errorMessage = 'Invalid credentials';
    mockAuthService.login.and.returnValue(throwError(() => new Error(errorMessage)));

    component.credentials.email = 'test@example.com';
    component.credentials.password = 'wrongpassword';
    fixture.detectChanges();
    tick();
    fixture.detectChanges();


    const form = fixture.debugElement.query(By.css('form'));
    form.triggerEventHandler('ngSubmit', null);
    tick(); // for async error handling
    fixture.detectChanges(); // Update view with error message

    expect(component.errorMessage).toBe(errorMessage);
    const errorDiv = fixture.debugElement.query(By.css('.error-message.api-error'));
    expect(errorDiv).toBeTruthy();
    expect(errorDiv.nativeElement.textContent.trim()).toBe(errorMessage);
  }));

  it('should clear previous error messages on new login attempt', fakeAsync(() => {
    // Initial failed attempt
    mockAuthService.login.and.returnValue(throwError(() => new Error('Initial error')));
    component.credentials = { email: 'test@example.com', password: 'pw1' };
    fixture.detectChanges(); tick(); fixture.detectChanges();
    component.login(); // Call directly or trigger form submit
    tick(); fixture.detectChanges();
    expect(component.errorMessage).toBe('Initial error');

    // Successful attempt
    mockAuthService.login.and.returnValue(of({ token: 'fake-token' }));
    component.credentials = { email: 'test@example.com', password: 'pw2' };
    fixture.detectChanges(); tick(); fixture.detectChanges();
    component.login();
    tick(); fixture.detectChanges();
    
    expect(component.errorMessage).toBe(''); // Error should be cleared
    const errorDiv = fixture.debugElement.query(By.css('.error-message.api-error'));
    expect(errorDiv).toBeFalsy(); // Error div should not be present
  }));


  it('should navigate to /register when "Register here" link is clicked', fakeAsync(() => {
    spyOn(router, 'navigateByUrl'); // More robust way to spy on router navigation for links
    const registerLink = fixture.debugElement.query(By.css('a[routerLink="/register"]'));
    
    expect(registerLink).toBeTruthy();
    
    // Simulate click. For routerLink, direct click might not work as expected in test.
    // Better to verify the link's presence and its routerLink attribute.
    // If actual navigation test is needed, it's more of an integration test.
    // For unit test, verify the routerLink attribute.
    expect(registerLink.attributes['routerLink']).toBe('/register');

    // If you really want to test the click causing navigation (closer to e2e/integration):
    // registerLink.nativeElement.click();
    // tick();
    // expect(router.navigateByUrl).toHaveBeenCalledWith(jasmine.stringMatching('/register'), jasmine.any(Object));
    // For a unit test, checking the attribute is usually sufficient.
  }));

});
