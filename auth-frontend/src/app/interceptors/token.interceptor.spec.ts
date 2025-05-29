import { TestBed } from '@angular/core/testing';
import { HttpClient, HTTP_INTERCEPTORS, HttpErrorResponse } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';

import { TokenInterceptor } from './token-interceptor'; // Use correct path
import { AuthService } from '../services/auth'; // Use correct path
import { environment } from '../../environments/environment';

describe('TokenInterceptor', () => {
  let httpClient: HttpClient;
  let httpMock: HttpTestingController;
  let mockAuthService: jasmine.SpyObj<AuthService>;

  const apiUrl = environment.apiUrl; // Base API URL from environment

  beforeEach(() => {
    mockAuthService = jasmine.createSpyObj('AuthService', ['getToken']);

    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
      providers: [
        TokenInterceptor,
        { provide: AuthService, useValue: mockAuthService },
        {
          provide: HTTP_INTERCEPTORS,
          useClass: TokenInterceptor,
          multi: true,
        },
      ],
    });

    httpClient = TestBed.inject(HttpClient);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify(); // Make sure that there are no outstanding requests.
  });

  it('should add an Authorization header when a token is available and request is to API URL', () => {
    const testToken = 'test-jwt-token';
    mockAuthService.getToken.and.returnValue(testToken);

    httpClient.get(`${apiUrl}/protected/data`).subscribe(response => {
      expect(response).toBeTruthy(); // Or any specific check on response
    });

    const httpRequest = httpMock.expectOne(`${apiUrl}/protected/data`);
    expect(httpRequest.request.headers.has('Authorization')).toBeTrue();
    expect(httpRequest.request.headers.get('Authorization')).toBe(`Bearer ${testToken}`);
    httpRequest.flush({ data: 'success' }); // Mock a response
  });

  it('should NOT add an Authorization header if no token is available', () => {
    mockAuthService.getToken.and.returnValue(null);

    httpClient.get(`${apiUrl}/protected/data`).subscribe(response => {
      expect(response).toBeTruthy();
    });

    const httpRequest = httpMock.expectOne(`${apiUrl}/protected/data`);
    expect(httpRequest.request.headers.has('Authorization')).toBeFalse();
    httpRequest.flush({ data: 'success' });
  });

  it('should NOT add an Authorization header for requests NOT to the API URL', () => {
    const testToken = 'test-jwt-token';
    mockAuthService.getToken.and.returnValue(testToken);
    const externalUrl = 'https://api.otherexample.com/data';

    httpClient.get(externalUrl).subscribe(response => {
      expect(response).toBeTruthy();
    });

    const httpRequest = httpMock.expectOne(externalUrl);
    expect(httpRequest.request.headers.has('Authorization')).toBeFalse();
    httpRequest.flush({ data: 'success' });
  });

  it('should NOT add an Authorization header for /auth/login requests', () => {
    const testToken = 'test-jwt-token';
    mockAuthService.getToken.and.returnValue(testToken);
    const loginUrl = `${apiUrl}/auth/login`;

    httpClient.post(loginUrl, {}).subscribe(response => {
      expect(response).toBeTruthy();
    });

    const httpRequest = httpMock.expectOne(loginUrl);
    expect(httpRequest.request.headers.has('Authorization')).toBeFalse();
    httpRequest.flush({ token: 'new-token' });
  });

  it('should NOT add an Authorization header for /auth/register requests', () => {
    const testToken = 'test-jwt-token';
    mockAuthService.getToken.and.returnValue(testToken);
    const registerUrl = `${apiUrl}/auth/register`;

    httpClient.post(registerUrl, {}).subscribe(response => {
      expect(response).toBeTruthy();
    });

    const httpRequest = httpMock.expectOne(registerUrl);
    expect(httpRequest.request.headers.has('Authorization')).toBeFalse();
    httpRequest.flush({ id: '123' });
  });
  
  it('should pass through the request if an error occurs in interceptor (though this interceptor does not throw errors)', () => {
    // This specific interceptor doesn't have logic that would throw an error itself.
    // If it did (e.g., trying to parse a malformed token from authService), this test would be relevant.
    // For now, it's more of a conceptual test for interceptor error handling.
    mockAuthService.getToken.and.returnValue('test-token');

    httpClient.get(`${apiUrl}/some/path`).subscribe(
        response => expect(response).toBeTruthy(),
        (error: HttpErrorResponse) => fail('Should not error in this basic pass-through scenario unless backend fails')
    );
    
    const req = httpMock.expectOne(`${apiUrl}/some/path`);
    req.flush({data: 'success'}); // Simulate a successful backend response
    // If the interceptor itself had an error, it would be caught by Angular's error handling or the test would fail.
  });

});
