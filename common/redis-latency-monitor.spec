################################################################################

# rpmbuilder:relative-pack true

################################################################################

%define  debug_package %{nil}

################################################################################

Summary:         Tiny Redis client for latency measurement
Name:            redis-latency-monitor
Version:         3.2.11
Release:         0%{?dist}
Group:           Applications/System
License:         Apache License, Version 2.0
URL:             https://kaos.sh/redis-latency-monitor

Source0:         https://source.kaos.st/%{name}/%{name}-%{version}.tar.bz2

BuildRoot:       %{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

BuildRequires:   golang >= 1.17

Provides:        %{name} = %{version}-%{release}

################################################################################

%description
Tiny Redis client for latency measurement. Utility show PING command latency
or connection latency in milliseconds (one thousandth of a second).

################################################################################

%prep
%setup -q

%build
export GOPATH=$(pwd)
go build src/github.com/essentialkaos/%{name}/%{name}.go

%install
rm -rf %{buildroot}

install -dm 755 %{buildroot}%{_bindir}
install -pm 755 %{name} %{buildroot}%{_bindir}/

%clean
rm -rf %{buildroot}

%post
if [[ -d %{_sysconfdir}/bash_completion.d ]] ; then
  %{name} --completion=bash 1> %{_sysconfdir}/bash_completion.d/%{name} 2>/dev/null
fi

if [[ -d %{_datarootdir}/fish/vendor_completions.d ]] ; then
  %{name} --completion=fish 1> %{_datarootdir}/fish/vendor_completions.d/%{name}.fish 2>/dev/null
fi

if [[ -d %{_datadir}/zsh/site-functions ]] ; then
  %{name} --completion=zsh 1> %{_datadir}/zsh/site-functions/_%{name} 2>/dev/null
fi

%postun
if [[ $1 == 0 ]] ; then
  if [[ -f %{_sysconfdir}/bash_completion.d/%{name} ]] ; then
    rm -f %{_sysconfdir}/bash_completion.d/%{name} &>/dev/null || :
  fi

  if [[ -f %{_datarootdir}/fish/vendor_completions.d/%{name}.fish ]] ; then
    rm -f %{_datarootdir}/fish/vendor_completions.d/%{name}.fish &>/dev/null || :
  fi

  if [[ -f %{_datadir}/zsh/site-functions/_%{name} ]] ; then
    rm -f %{_datadir}/zsh/site-functions/_%{name} &>/dev/null || :
  fi
fi

################################################################################

%files
%defattr(-,root,root,-)
%doc LICENSE
%{_bindir}/%{name}

################################################################################

%changelog
* Wed Mar 30 2022 Anton Novojilov <andy@essentialkaos.com> - 3.2.1-0
- Package ek updated to the latest stable version
- Removed pkg.re usage
- Added module info
- Added Dependabot configuration

* Mon Oct 21 2019 Anton Novojilov <andy@essentialkaos.com> - 3.2.0-0
- Improved UI

* Thu Oct 17 2019 Anton Novojilov <andy@essentialkaos.com> - 3.1.1-0
- ek package updated to the latest stable version

* Thu Jun 13 2019 Anton Novojilov <andy@essentialkaos.com> - 3.1.0-0
- ek package updated to the latest stable version
- Added completion generation for bash, zsh and fish

* Wed Oct 31 2018 Anton Novojilov <andy@essentialkaos.com> - 3.0.3-0
- Fixed bug with Max/Mean/StDev/Perc calculation
- Minor UI improvements

* Sat Oct 20 2018 Anton Novojilov <andy@essentialkaos.com> - 3.0.2-0
- Show usage info if '-h' passed without any value

* Thu Dec 21 2017 Anton Novojilov <andy@essentialkaos.com> - 3.0.1-0
- Minor UI fixes

* Tue Dec 19 2017 Anton Novojilov <andy@essentialkaos.com> - 3.0.0-0
- Fixed bug with percentile calculation
- ek package updated to latest version
- More precise latency calculation
- Removed external packages
- Improved UI
- Code refactoring

* Tue Oct 03 2017 Anton Novojilov <andy@essentialkaos.com> - 2.4.0-0
- Added option -T/--timestamps for output time as unix timestamp

* Thu Jul 06 2017 Anton Novojilov <andy@essentialkaos.com> - 2.3.1-0
- Added auth error handling

* Mon Jun 26 2017 Anton Novojilov <andy@essentialkaos.com> - 2.3.0-0
- Added option --error-log/-e for logging error messages
- Improved working with Redis connection

* Fri Jun 23 2017 Anton Novojilov <andy@essentialkaos.com> - 2.2.0-0
- Alignment of interval start point

* Fri Jun 16 2017 Anton Novojilov <andy@essentialkaos.com> - 2.1.0-0
- Improved UI

* Fri Jun 16 2017 Anton Novojilov <andy@essentialkaos.com> - 2.0.0-0
- Connection latency measurement
- Output measurements in CSV format

* Wed Jun 14 2017 Anton Novojilov <andy@essentialkaos.com> - 1.1.0-0
- Measurements slice reusage
- Improved UI and log output

* Fri Jun 09 2017 Anton Novojilov <andy@essentialkaos.com> - 1.0.2-0
- Improved UI

* Thu Jun 08 2017 Anton Novojilov <andy@essentialkaos.com> - 1.0.1-0
- Minor improvements

* Wed Jun 07 2017 Anton Novojilov <andy@essentialkaos.com> - 1.0.0-0
- Initial build
